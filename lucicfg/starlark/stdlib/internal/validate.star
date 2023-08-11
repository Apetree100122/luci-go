# Copyright 2018 The LUCI Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

"""Generic value validators."""

load("@stdlib//internal/lucicfg.star", "lucicfg")
load("@stdlib//internal/re.star", "re")
load("@stdlib//internal/time.star", "time")

def _string(attr, val, *, regexp = None, allow_empty = False, default = None, required = True):
    """Validates that the value is a string and returns it.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      regexp: a regular expression to check 'val' against.
      allow_empty: if True, accept empty string as valid.
      default: a value to use if 'val' is None, ignored if required is True.
      required: if False, allow 'val' to be None, return 'default' in this case.

    Returns:
      The validated string or None if required is False and default is None.
    """
    if val == None:
        if required:
            fail("missing required field %r" % attr)
        if default == None:
            return None
        val = default

    if type(val) != "string":
        fail("bad %r: got %s, want string" % (attr, type(val)))
    if not allow_empty and not val:
        fail("bad %r: must not be empty" % (attr,))
    if regexp and not re.submatches(regexp, val):
        fail("bad %r: %r should match %r" % (attr, val, regexp))

    return val

def _hostname(attr, val, *, default = None, required = True):
    """Validates that the value is a string RFC 1123 hostname and returns it.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      default: a value to use if 'val' is None, ignored if required is True.
      required: if False, allow 'val' to be None, return 'default' in this case.

    Returns:
      The validated hostname or None if required is False and default is None.
    """
    if val == None:
        if required:
            fail("missing required field %r" % attr)
        if default == None:
            return None
        val = default

    if type(val) != "string":
        fail("bad %r: got %s, want string" % (attr, type(val)))
    if not val:
        fail("bad %r: must not be empty" % (attr,))

    hostname_regexp = r"^(?:(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])\.)*(?:[A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9-]*[A-Za-z0-9])$"
    if not re.submatches(hostname_regexp, val):
        fail("bad %r: %r is not valid RFC1123 hostname" % (attr, val))

    return val

def _int(attr, val, *, min = None, max = None, default = None, required = True):
    """Validates that the value is an integer and returns it.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      min: minimal allowed value (inclusive) or None for unbounded.
      max: maximal allowed value (inclusive) or None for unbounded.
      default: a value to use if 'val' is None, ignored if required is True.
      required: if False, allow 'val' to be None, return 'default' in this case.

    Returns:
      The validated int or None if required is False and default is None.
    """
    if val == None:
        if required:
            fail("missing required field %r" % attr)
        if default == None:
            return None
        val = default

    if type(val) != "int":
        fail("bad %r: got %s, want int" % (attr, type(val)))

    if min != None and val < min:
        fail("bad %r: %s should be >= %s" % (attr, val, min))
    if max != None and val > max:
        fail("bad %r: %s should be <= %s" % (attr, val, max))

    return val

def _float(attr, val, *, min = None, max = None, default = None, required = True):
    """Validates that the value is a float or integer and returns it as float.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      min: minimal allowed value (inclusive) or None for unbounded.
      max: maximal allowed value (inclusive) or None for unbounded.
      default: a value to use if 'val' is None, ignored if required is True.
      required: if False, allow 'val' to be None, return 'default' in this case.

    Returns:
      The validated float or None if required is False and default is None.
    """
    if val == None:
        if required:
            fail("missing required field %r" % attr)
        if default == None:
            return None
        val = default

    if type(val) == "int":
        val = float(val)
    elif type(val) != "float":
        fail("bad %r: got %s, want float or int" % (attr, type(val)))

    if min != None and val < min:
        fail("bad %r: %s should be >= %s" % (attr, val, min))
    if max != None and val > max:
        fail("bad %r: %s should be <= %s" % (attr, val, max))

    return val

def _bool(attr, val, *, default = None, required = True):
    """Validates that the value can be converted to a boolean.

    Zero values other than None (0, "", [], etc) are treated as False. None
    indicates "use default". If required is False and val is None, returns None
    (indicating no value was passed).

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      default: a value to use if 'val' is None, ignored if required is True.
      required: if False, allow 'val' to be None, return 'default' in this case.

    Returns:
      The boolean or None if required is False and default is None.
    """
    if val == None:
        if required:
            fail("missing required field %r" % attr)
        if default == None:
            return None
        val = default
    return bool(val)

def _duration(attr, val, *, precision = time.second, min = time.zero, max = None, default = None, required = True):
    """Validates that the value is a duration specified at the given precision.

    For example, if 'precision' is time.second, will validate that the given
    duration has a whole number of seconds. Fails if truncating the duration to
    the requested precision loses information.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      precision: a time unit to divide 'val' by to get the output.
      min: minimal allowed duration (inclusive) or None for unbounded.
      max: maximal allowed duration (inclusive) or None for unbounded.
      default: a value to use if 'val' is None, ignored if required is True.
      required: if False, allow 'val' to be None, return 'default' in this case.

    Returns:
      The validated duration or None if required is False and default is None.
    """
    if val == None:
        if required:
            fail("missing required field %r" % attr)
        if default == None:
            return None
        val = default

    if type(val) != "duration":
        fail("bad %r: got %s, want duration" % (attr, type(val)))

    if min != None and val < min:
        fail("bad %r: %s should be >= %s" % (attr, val, min))
    if max != None and val > max:
        fail("bad %r: %s should be <= %s" % (attr, val, max))

    if time.truncate(val, precision) != val:
        fail((
            "bad %r: losing precision when truncating %s to %s units, " +
            "use time.truncate(...) to acknowledge"
        ) % (attr, val, precision))

    return val

def _email(attr, val, *, default = None, required = True):
    """Validates that the value is a string RFC 2822 hostname and returns it.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      default: a value to use if 'val' is None, ignored if required is True.
      required: if False, allow 'val' to be None, return 'default' in this case.

    Returns:
      The validated email or None if required is False and default is None.
    """
    if val == None:
        if required:
            fail("missing required field %r" % attr)
        if default == None:
            return None
        val = default

    if type(val) != "string":
        fail("bad %r: got %s, want string" % (attr, type(val)))
    if not val:
        fail("bad %r: must not be empty" % (attr,))

    email_regexp = r"^[a-z0-9!#$%&'*+/=?^_`{|}~-]+(?:\.[a-z0-9!#$%&'*+/=?^_`{|}~-]+)*@(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$"
    if not re.submatches(email_regexp, val.lower()):
        fail("bad %r: %r is not a valid RFC 2822 email" % (attr, val))

    return val

def _list(attr, val, *, required = False):
    """Validates that the value is a list and returns it.

    None is treated as an empty list.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      required: if False, allow 'val' to be None or empty, return empty list in
        this case.

    Returns:
      The validated list.
    """
    if val == None:
        val = []

    if type(val) != "list":
        fail("bad %r: got %s, want list" % (attr, type(val)))

    if required and not val:
        fail("missing required field %r" % attr)

    return val

def _str_dict(attr, val, *, required = False):
    """Validates that the value is a dict with non-empty string keys.

    None is treated as an empty dict.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      required: if False, allow 'val' to be None or empty, return empty dict in
        this case.

    Returns:
      The validated dict.
    """
    if val == None:
        val = {}

    if type(val) != "dict":
        fail("bad %r: got %s, want dict" % (attr, type(val)))

    if required and not val:
        fail("missing required field %r" % attr)

    for k in val:
        if type(k) != "string":
            fail("bad %r: got %s key, want string" % (attr, type(k)))
        if not k:
            fail("bad %r: got empty key" % attr)

    return val

def _struct(attr, val, sym, *, default = None, required = True):
    """Validates that the value is a struct of the given flavor and returns it.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      sym: a name of the constructor that produced the struct.
      default: a value to use if 'val' is None, ignored if required is True.
      required: if False, allow 'val' to be None, return 'default' in this case.

    Returns:
      The validated struct or None if required is False and default is None.
    """
    if val == None:
        if required:
            fail("missing required field %r" % attr)
        if default == None:
            return None
        val = default

    tp = __native__.ctor(val) or type(val)  # ctor(...) return None for non-structs
    if tp != sym:
        fail("bad %r: got %s, want %s" % (attr, tp, sym))

    return val

def _type(attr, val, prototype, *, default = None, required = True):
    """Validates the value has the same type as `prototype` or is None.

    Useful when checking types of protobuf messages.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      prototype: a prototype value to compare val's type against.
      default: a value to use if `val` is None, ignored if required is True.
      required: if False, allow `val` to be None, return `default` in this case.

    Returns:
      `val` on success or None if required is False and default is None.
    """
    if val == None:
        if required:
            fail("missing required field %r" % attr)
        if default == None:
            return None
        val = default

    if type(val) != type(prototype):
        fail("bad %r: got %s, want %s" % (attr, type(val), type(prototype)))

    return val

def _repo_url(attr, val, *, required = True):
    """Validates that the value is `https://...` repository URL and returns it.

    Additionally verifies that `val` doesn't end with `.git`.

    Args:
      attr: name of the var for error messages. Required.
      val: a value to validate. Required.
      required: if False, allow `val` to be None, return None in this case.

    Returns:
      Validate `val` or None if it is None and `required` is False.
    """
    val = validate.string(attr, val, regexp = r"https://.+", required = required)
    if val and val.endswith(".git"):
        fail('bad %r: %r should not end with ".git"' % (attr, val))
    return val

def _relative_path(attr, val, *, allow_dots = False, base = None, required = True, default = None):
    """Validates that the value is a string with relative path.

    Optionally adds it to some base path and returns the cleaned resulting path.

    Args:
      attr: name of the var for error messages. Required.
      val: a value to validate. Required.
      allow_dots: if True, allow `../` as a prefix in the resulting path.
        Default is False.
      base: if given, apply the relative path to this base path and returns the
        result.
      default: a value to use if 'val' is None, ignored if required is True.
      required: if False, allow 'val' to be None, return `default` in this case.

    Returns:
      Validated, cleaned and (if `base` is given) rebased path.
    """
    val = validate.string(attr, val, required = required, default = default)
    if val == None:
        return None
    base = validate.string("base", base, required = False)
    clean, err = __native__.clean_relative_path(base or "", val, allow_dots)
    if err:
        fail("bad %r: %s" % (attr, err))
    return clean

def _regex_list(attr, val, *, required = False):
    """Validates that the value is a valid regex parameter.

    Strings are valid, and are returned unchanged. Lists of strings are valid,
    and are combined into a single regex that matches any of the regexes in
    the list.

    None is treated as an empty string.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      required: if False, allow 'val' to be None or empty, return empty string
        in this case.

    Returns:
      The validated regex.
    """
    if val == None:
        val = ""

    if required and not val:
        fail("missing required field %r" % attr)

    if type(val) == "string":
        valid, err = __native__.is_valid_regex(val)
        if not valid:
            fail("bad %r: %s" % (attr, err))
        return val
    if type(val) == "list":
        # buildifier: disable=string-iteration
        for s in val:
            if type(s) != "string":
                fail("bad %r: got list element of type %s, want string" % (attr, type(s)))
            valid, err = __native__.is_valid_regex(s)
            if not valid:
                fail("bad %r: %s" % (attr, err))
        return "|".join(val)

    fail("bad %r: got %s, want string or list" % (attr, type(val)))

def _str_list(attr, val, *, required = False):
    """Validates that the value is a list of strings.

    None is treated as an empty list.

    Args:
      attr: field name with this value, for error messages.
      val: a value to validate.
      required: if False, allow 'val' to be None or empty, return empty list in
        this case.

    Returns:
      The validated list.
    """
    if val == None:
        val = []

    if required and not val:
        fail("missing required field %r" % attr)

    if type(val) == "list":
        for s in val:
            if type(s) != "string":
                fail("bad %r: got list element of type %s, want string" % (attr, type(s)))
        return val

    fail("bad %r: got %s, want list of strings" % (attr, type(val)))

def _var_with_validator(attr, validator, **kwargs):
    """Returns a lucicfg.var that validates the value via a validator callback.

    Args:
      attr: name of the var for error messages. Required.
      validator: a callback(attr, value, **kwargs), e.g. `validate.string`.
        Required.
      **kwargs: keyword arguments to pass to `validator`.

    Returns:
      lucicfg.var(...).
    """
    return lucicfg.var(validator = lambda value: validator(attr, value, **kwargs))

def _vars_with_validators(vars):
    """Accepts dict `{attr -> validator}`, returns dict `{attr -> lucicfg.var}`.

    Basically applies validate.var_with_validator(...) to each item of the dict.

    Args:
      vars: a dict with string keys and callable values, matching the signature
        of `validator` in validate.var_with_validator(...). Required.

    Returns:
      Dict with string keys and lucicfg.var(...) values.
    """
    return {attr: _var_with_validator(attr, validator) for attr, validator in vars.items()}

validate = struct(
    string = _string,
    int = _int,
    float = _float,
    bool = _bool,
    duration = _duration,
    email = _email,
    list = _list,
    str_dict = _str_dict,
    struct = _struct,
    type = _type,
    repo_url = _repo_url,
    hostname = _hostname,
    relative_path = _relative_path,
    regex_list = _regex_list,
    str_list = _str_list,
    var_with_validator = _var_with_validator,
    vars_with_validators = _vars_with_validators,
)
