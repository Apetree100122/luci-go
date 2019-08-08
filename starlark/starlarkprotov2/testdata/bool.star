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

l = proto.new_loader(read('./testprotos/all.pb'))
testprotos = l.module('go.chromium.org/luci/starlark/starlarkprotov2/testprotos/test.proto')

m = testprotos.SimpleFields()

# Default value.
assert.eq(m.b, False)

# Setter and getter works.
m.b = True
assert.eq(m.b, True)
assert.eq(proto.to_textpb(m), 'b: true\n')
m.b = False
assert.eq(m.b, False)
assert.eq(proto.to_textpb(m), '\n')  # 'false' is default!

# Setting through constructor works.
m2 = testprotos.SimpleFields(b=True)
assert.eq(m2.b, True)

# Clearing works.
m2.b = None
assert.eq(m2.b, False)

# Setting wrong type fails.
def set_bad():
  m2.b = [1, 2, 3]
assert.fails(set_bad, 'can\'t assign "list" to "bool" field')

# We don't support implicit conversions to bool. Callers should use bool(...)
# cast explicitly.
def set_int():
  m2.b = 0
assert.fails(set_int, 'can\'t assign "int" to "bool" field')

# Assiging bool to non-bool field fails.
def set_bool_to_int():
  m2.i64 = False
assert.fails(set_bool_to_int, 'can\'t assign "bool" to "int64" field')
