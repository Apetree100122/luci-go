// Copyright 2017 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package notify

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"

	"go.chromium.org/gae/service/datastore"
	"go.chromium.org/gae/service/info"
	"go.chromium.org/gae/service/mail"
	"go.chromium.org/luci/appengine/tq"
	buildbucketpb "go.chromium.org/luci/buildbucket/proto"
	"go.chromium.org/luci/common/data/stringset"
	"go.chromium.org/luci/common/errors"
	"go.chromium.org/luci/common/logging"
	gitpb "go.chromium.org/luci/common/proto/git"
	"go.chromium.org/luci/common/sync/parallel"
	"go.chromium.org/luci/server/auth"

	notifypb "go.chromium.org/luci/luci_notify/api/config"
	"go.chromium.org/luci/luci_notify/config"
	"go.chromium.org/luci/luci_notify/internal"
	"go.chromium.org/luci/luci_notify/mailtmpl"
)

// createEmailTasks constructs EmailTasks to be dispatched onto the task queue.
func createEmailTasks(c context.Context, recipients []EmailNotify, input *notifypb.TemplateInput) ([]*tq.Task, error) {
	// Get templates.
	bundle, err := getBundle(c, input.Build.Builder.Project)
	if err != nil {
		return nil, errors.Annotate(err, "failed to get a bundle of email templates").Err()
	}

	// Generate emails.
	// An EmailTask with subject and body per template name.
	// They will be used as templates for actual tasks.
	taskTemplates := map[string]*internal.EmailTask{}
	for _, r := range recipients {
		name := r.Template
		if name == "" {
			name = mailtmpl.DefaultTemplateName
		}

		if _, ok := taskTemplates[name]; ok {
			continue
		}
		input.MatchingFailedSteps = r.MatchingSteps

		subject, body := bundle.GenerateEmail(name, input)

		// Note: this buffer should not be reused.
		buf := &bytes.Buffer{}
		gz := gzip.NewWriter(buf)
		io.WriteString(gz, body)
		if err := gz.Close(); err != nil {
			panic("failed to gzip HTML body in memory")
		}
		taskTemplates[name] = &internal.EmailTask{
			Subject:  subject,
			BodyGzip: buf.Bytes(),
		}
	}

	// Create a task per recipient.
	// Do not bundle multiple recipients into one task because we don't use BCC.
	tasks := make([]*tq.Task, 0, len(recipients))
	seen := stringset.New(len(recipients))
	for _, r := range recipients {
		name := r.Template
		if name == "" {
			name = mailtmpl.DefaultTemplateName
		}

		emailKey := fmt.Sprintf("%d-%s-%s", input.Build.Id, name, r.Email)
		if seen.Has(emailKey) {
			continue
		}
		seen.Add(emailKey)

		task := *taskTemplates[name] // copy
		task.Recipients = []string{r.Email}
		tasks = append(tasks, &tq.Task{
			DeduplicationKey: emailKey,
			Payload:          &task,
		})
	}
	return tasks, nil
}

// isRecipientAllowed returns true if the given recipient is allowed to be notified about the given build.
func isRecipientAllowed(c context.Context, recipient string, build *buildbucketpb.Build) bool {
	// TODO(mknyszek): Do a real ACL check here.
	if strings.HasSuffix(recipient, "@google.com") || strings.HasSuffix(recipient, "@chromium.org") {
		return true
	}
	logging.Warningf(c, "Address %q is not allowed to be notified of build %d", recipient, build.Id)
	return false
}

// BlamelistRepoWhiteset computes the aggregate repository whitelist for all
// blamelist notification configurations in a given set of notifications.
func BlamelistRepoWhiteset(notifications notifypb.Notifications) stringset.Set {
	whiteset := stringset.New(0)
	for _, notification := range notifications.GetNotifications() {
		blamelistInfo := notification.GetNotifyBlamelist()
		for _, repo := range blamelistInfo.GetRepositoryWhitelist() {
			whiteset.Add(repo)
		}
	}
	return whiteset
}

// ToNotify encapsulates a notification, along with the list of matching steps
// necessary to render templates for that notification. It's used to pass this
// data between the filtering/matching code and the code responsible for sending
// emails and updating tree status.
type ToNotify struct {
	Notification  *notifypb.Notification
	MatchingSteps []*buildbucketpb.Step
}

// ComputeRecipients computes the set of recipients given a set of
// notifications, and potentially "input" and "output" blamelists.
//
// An "input" blamelist is computed from the input commit to a build, while an
// "output" blamelist is derived from output commits.
func ComputeRecipients(c context.Context, notifications []ToNotify, inputBlame []*gitpb.Commit, outputBlame Logs) []EmailNotify {
	return computeRecipientsInternal(c, notifications, inputBlame, outputBlame,
		func(c context.Context, url string) ([]byte, error) {
			transport, err := auth.GetRPCTransport(c, auth.AsSelf)
			if err != nil {
				return nil, err
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return nil, err
			}
			req = req.WithContext(c)

			response, err := (&http.Client{Transport: transport}).Do(req)
			if err != nil {
				return nil, errors.Annotate(err, "failed to get data from %q", url).Err()
			}

			defer response.Body.Close()
			bytes, err := ioutil.ReadAll(response.Body)
			if err != nil {
				return nil, errors.Annotate(err, "failed to read response body from %q", url).Err()
			}

			return bytes, nil
		})
}

// computeRecipientsInternal also takes fetchFunc, so http requests can be
// mocked out for testing.
func computeRecipientsInternal(c context.Context, notifications []ToNotify, inputBlame []*gitpb.Commit, outputBlame Logs, fetchFunc func(context.Context, string) ([]byte, error)) []EmailNotify {
	recipients := make([]EmailNotify, 0)
	for _, toNotify := range notifications {
		appendRecipient := func(e EmailNotify) {
			e.MatchingSteps = toNotify.MatchingSteps
			recipients = append(recipients, e)
		}

		n := toNotify.Notification

		// Aggregate the static list of recipients from the Notifications.
		for _, recipient := range n.GetEmail().GetRecipients() {
			appendRecipient(EmailNotify{
				Email:    recipient,
				Template: n.Template,
			})
		}

		// Don't bother dealing with anything blamelist related if there's no config for it.
		if n.NotifyBlamelist == nil {
			continue
		}

		// If the whitelist is empty, use the static blamelist.
		whitelist := n.NotifyBlamelist.GetRepositoryWhitelist()
		if len(whitelist) == 0 {
			for _, e := range commitsBlamelist(inputBlame, n.Template) {
				appendRecipient(e)
			}
			continue
		}

		// If the whitelist is non-empty, use the dynamic blamelist.
		whiteset := stringset.NewFromSlice(whitelist...)
		for _, e := range outputBlame.Filter(whiteset).Blamelist(n.Template) {
			appendRecipient(e)
		}
	}

	// Acquired before appending to "recipients" from the tasks below.
	mRecipients := sync.Mutex{}
	err := parallel.WorkPool(8, func(ch chan<- func() error) {
		for _, toNotify := range notifications {
			template := toNotify.Notification.Template
			steps := toNotify.MatchingSteps
			for _, rotation := range toNotify.Notification.GetEmail().GetRotaNgRotations() {
				rotation := rotation
				ch <- func() error {
					return fetchRotaOncallers(c, rotation, template, steps, fetchFunc, &recipients, &mRecipients)
				}
			}
		}
	})

	if err != nil {
		// Just log the error and continue. Nothing much else we can do, and it's possible that we only failed
		// to fetch some of the recipients, so we can at least return the ones we were able to compute.
		logging.Errorf(c, "failed to fetch some or all Rota-NG oncallers: %s", err)
	}

	return recipients
}

func fetchRotaOncallers(c context.Context, rotation, template string, matchingSteps []*buildbucketpb.Step, fetchFunc func(context.Context, string) ([]byte, error), recipients *[]EmailNotify, mRecipients *sync.Mutex) error {
	address := "https://rota-ng.appspot.com/legacy/" + url.PathEscape(rotation) + ".json"
	resp, err := fetchFunc(c, address)
	if err != nil {
		err = errors.Annotate(err, "failed to fetch address %q", address).Err()
		return err
	}

	var oncallEmails struct {
		Emails []string
	}
	if err = json.Unmarshal(resp, &oncallEmails); err != nil {
		return errors.Annotate(err, "failed to unmarshal JSON").Err()
	}

	mRecipients.Lock()
	defer mRecipients.Unlock()
	for _, email := range oncallEmails.Emails {
		*recipients = append(*recipients, EmailNotify{
			Email:         email,
			Template:      template,
			MatchingSteps: matchingSteps,
		})
	}

	return nil
}

// ShouldNotify determines whether a trigger's conditions have been met, and returns the list
// of steps matching the filters on the notification, if any.
func ShouldNotify(n *notifypb.Notification, oldStatus buildbucketpb.Status, newBuild *buildbucketpb.Build) (bool, []*buildbucketpb.Step) {
	newStatus := newBuild.Status

	switch {

	case newStatus == buildbucketpb.Status_STATUS_UNSPECIFIED:
		panic("new status must always be valid")
	case contains(newStatus, n.OnOccurrence):
	case oldStatus != buildbucketpb.Status_STATUS_UNSPECIFIED && newStatus != oldStatus && contains(newStatus, n.OnNewStatus):

	// deprecated functionality
	case n.OnSuccess && newStatus == buildbucketpb.Status_SUCCESS:
	case n.OnFailure && newStatus == buildbucketpb.Status_FAILURE:
	case n.OnChange && oldStatus != buildbucketpb.Status_STATUS_UNSPECIFIED && newStatus != oldStatus:
	case n.OnNewFailure && newStatus == buildbucketpb.Status_FAILURE && oldStatus != buildbucketpb.Status_FAILURE:

	default:
		return false, nil
	}

	return matchingSteps(newBuild, n.FailedStepRegexp, n.FailedStepRegexpExclude)
}

func matchingSteps(build *buildbucketpb.Build, failedStepRegexp, failedStepRegexpExclude string) (bool, []*buildbucketpb.Step) {
	var includeRegex *regexp.Regexp
	if failedStepRegexp != "" {
		// We should never get an invalid regex here, as our validation should catch this.
		includeRegex = regexp.MustCompile(fmt.Sprintf("^%s$", failedStepRegexp))
	}

	var excludeRegex *regexp.Regexp
	if failedStepRegexpExclude != "" {
		// Ditto.
		excludeRegex = regexp.MustCompile(fmt.Sprintf("^%s$", failedStepRegexpExclude))
	}

	// Don't scan steps if we have no regexes.
	if includeRegex == nil && excludeRegex == nil {
		return true, nil
	}

	var steps []*buildbucketpb.Step
	for _, step := range build.Steps {
		if step.Status == buildbucketpb.Status_FAILURE {
			if (includeRegex == nil || includeRegex.MatchString(step.Name)) &&
				(excludeRegex == nil || !excludeRegex.MatchString(step.Name)) {
				steps = append(steps, step)
			}
		}
	}

	if len(steps) > 0 {
		return true, steps
	}
	return false, nil
}

// Filter filters out Notification objects from Notifications by checking if we ShouldNotify
// based on two build statuses.
func Filter(n *notifypb.Notifications, oldStatus buildbucketpb.Status, newBuild *buildbucketpb.Build) []ToNotify {
	notifications := n.GetNotifications()
	filtered := make([]ToNotify, 0, len(notifications))
	for _, notification := range notifications {
		if match, steps := ShouldNotify(notification, oldStatus, newBuild); match {
			filtered = append(filtered, ToNotify{
				Notification:  notification,
				MatchingSteps: steps,
			})
		}
	}
	return filtered
}

// contains checks whether or not a build status is in a list of build statuses.
func contains(status buildbucketpb.Status, statusList []buildbucketpb.Status) bool {
	for _, s := range statusList {
		if status == s {
			return true
		}
	}
	return false
}

// UpdateTreeClosers finds all the TreeClosers that care about a particular
// build, and updates their status according to the results of the build.
func UpdateTreeClosers(c context.Context, build *Build, oldStatus buildbucketpb.Status) error {
	// This reads, modifies and writes back entities in datastore. Hence, it should
	// be called within a transaction to avoid races.
	if datastore.CurrentTransaction(c) == nil {
		panic("UpdateTreeClosers must be run within a transaction")
	}

	// Don't update the status at all unless we have a definite
	// success or failure - infra failures, for example, shouldn't
	// cause us to close or re-open the tree.
	if build.Status != buildbucketpb.Status_SUCCESS && build.Status != buildbucketpb.Status_FAILURE {
		return nil
	}

	project := &config.Project{Name: build.Builder.Project}
	parentBuilder := &config.Builder{
		ProjectKey: datastore.KeyForObj(c, project),
		ID:         getBuilderID(&build.Build),
	}
	q := datastore.NewQuery("TreeCloser").Ancestor(datastore.KeyForObj(c, parentBuilder))
	var toUpdate []*config.TreeCloser
	if err := datastore.GetAll(c, q, &toUpdate); err != nil {
		return err
	}

	for _, tc := range toUpdate {
		newStatus := config.Open
		var steps []*buildbucketpb.Step
		if build.Status == buildbucketpb.Status_FAILURE {
			t := tc.TreeCloser
			var match bool
			if match, steps = matchingSteps(&build.Build, t.FailedStepRegexp, t.FailedStepRegexpExclude); match {
				newStatus = config.Closed
			}
		}

		tc.Status = newStatus
		var err error
		if tc.Timestamp, err = ptypes.Timestamp(build.EndTime); err != nil {
			logging.Warningf(c, "Build EndTime is invalid (%s), defaulting to time.Now()", err)
			tc.Timestamp = time.Now().UTC()
		}

		if newStatus == config.Closed {
			bundle, err := getBundle(c, project.Name)
			if err != nil {
				return err
			}
			tc.Message = bundle.GenerateStatusMessage(c, tc.TreeCloser.Template,
				&notifypb.TemplateInput{
					BuildbucketHostname: build.BuildbucketHostname,
					Build:               &build.Build,
					OldStatus:           oldStatus,
					MatchingFailedSteps: steps,
				})
		} else {
			// Not strictly necessary, as Message is only used when status is
			// 'Closed'. But it could be confusing when debugging to have a
			// stale message in the entity.
			tc.Message = ""
		}
	}

	return datastore.Put(c, toUpdate)
}

// Notify discovers, consolidates and filters recipients from a Builder's notifications,
// and 'email_notify' properties, then dispatches notifications if necessary.
// Does not dispatch a notification for same email, template and build more than
// once. Ignores current transaction in c, if any.
func Notify(c context.Context, d *tq.Dispatcher, recipients []EmailNotify, templateParams *notifypb.TemplateInput) error {
	c = datastore.WithoutTransaction(c)

	// Remove unallowed recipients.
	allRecipients := recipients
	recipients = recipients[:0]
	for _, r := range allRecipients {
		if isRecipientAllowed(c, r.Email, templateParams.Build) {
			recipients = append(recipients, r)
		}
	}

	if len(recipients) == 0 {
		logging.Infof(c, "Nobody to notify...")
		return nil
	}
	tasks, err := createEmailTasks(c, recipients, templateParams)
	if err != nil {
		return errors.Annotate(err, "failed to create email tasks").Err()
	}
	return d.AddTask(c, tasks...)
}

// InitDispatcher registers the send email task with the given dispatcher.
func InitDispatcher(d *tq.Dispatcher) {
	d.RegisterTask(&internal.EmailTask{}, SendEmail, "email", nil)
}

// SendEmail is a push queue handler that attempts to send an email.
func SendEmail(c context.Context, task proto.Message) error {
	appID := info.AppID(c)
	sender := fmt.Sprintf("%s <noreply@%s.appspotmail.com>", appID, appID)

	// TODO(mknyszek): Query Milo for additional build information.
	emailTask := task.(*internal.EmailTask)

	body := emailTask.Body
	if len(emailTask.BodyGzip) > 0 {
		r, err := gzip.NewReader(bytes.NewReader(emailTask.BodyGzip))
		if err != nil {
			return err
		}
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			return err
		}
		body = string(buf)
	}

	return mail.Send(c, &mail.Message{
		Sender:   sender,
		To:       emailTask.Recipients,
		Subject:  emailTask.Subject,
		HTMLBody: body,
		ReplyTo:  emailTask.Recipients[0],
	})
}
