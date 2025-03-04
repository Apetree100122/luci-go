// Copyright 2022 The LUCI Authors.
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

import { useParams } from 'react-router-dom';

import Container from '@mui/material/Container';
import Grid from '@mui/material/Grid';
import LinearProgress from '@mui/material/LinearProgress';

import ClusterAnalysisSection from '@/components/cluster/cluster_analysis_section/cluster_analysis_section';
import { ClusterContextProvider } from '@/components/cluster/cluster_context';
import ErrorAlert from '@/components/error_alert/error_alert';
import LoadErrorAlert from '@/components/load_error_alert/load_error_alert';
import RuleArchivedMessage from '@/components/rule/rule_archived_message/rule_archived_message';
import RuleTopPanel from '@/components/rule/rule_top_panel/rule_top_panel';
import useFetchRule from '@/hooks/use_fetch_rule';

const Rule = () => {
  const { project, id } = useParams();

  const { isLoading, data: rule, error, isSuccess } = useFetchRule(id, project);

  if (!project || !id) {
    return (
      <ErrorAlert
        errorTitle="Project or Rule ID is not specified"
        errorText="Project or Rule ID not specified in the URL, please make sure you have the correct URL and try again."
        showError
      />
    );
  }

  return (
    <Container className='mt-1' maxWidth={false}>
      <Grid sx={{ mt: 1 }} container spacing={2}>
        {isLoading && (
          <LinearProgress />
        )}
        {
          error && (
            <LoadErrorAlert
              entityName="rule"
              error={error}
            />
          )
        }
        {
          isSuccess && rule && (
            <>
              <Grid item xs={12}>
                <RuleTopPanel project={project} ruleId={id} />
              </Grid>
              <Grid item xs={12}>
                {rule?.isActive ?
                  (
                    <ClusterContextProvider project={project} clusterAlgorithm='rules' clusterId={id}>
                      <ClusterAnalysisSection />
                    </ClusterContextProvider>
                  ) : (
                    <RuleArchivedMessage />
                  )
                }
              </Grid>
            </>
          )
        }
      </Grid>
    </Container>
  );
};

export default Rule;
