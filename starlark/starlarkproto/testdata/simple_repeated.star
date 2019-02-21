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

load("go.chromium.org/luci/starlark/starlarkproto/testprotos/test.proto", "testprotos")

# Note: this test also covers all other scalar types, since the implementation
# of repeated fields is identical for all of them.

m = testprotos.SimpleFields()

# Default value.
assert.eq(m.i64_rep, [])

# Can append to it, it is just a list.
m.i64_rep.append(1)
assert.eq(m.i64_rep, [1])

# Can completely recreated the field by replacing with default.
m.i64_rep = None
assert.eq(m.i64_rep, [])

# The list is stored as a reference, not as a value.
l = []
m2 = testprotos.SimpleFields(i64_rep=l)
l.append(123)
assert.eq(m2.i64_rep, [123])

# Setter works. It preserves the reference and type of the value as long as it
# is iterable.

l2 = [1, 2]
m2.i64_rep = l2
assert.eq(m2.i64_rep, [1, 2])
l2.append(3)
assert.eq(m2.i64_rep, [1, 2, 3])

t = (1, 2)
m2.i64_rep = t
assert.eq(m2.i64_rep, (1, 2))

# Trying to set a wrong type is an error.
def set_int():
  m2.i64_rep = 123
assert.fails(set_int, 'can\'t assign integer to a value of kind "slice"')

# Sneakily adding wrong-typed element to the list is NOT an immediate error
# currently. This is discovered later when trying to serialize the object.
m2.i64_rep = [1, 2, None]
def serialize():
  proto.to_textpb(m2)
assert.fails(serialize, 'list item #2 - can\'t assign nil to a value of kind "int64"')

# Serialization to text proto works.
text = proto.to_textpb(testprotos.SimpleFields(i64_rep=[1, 2, 3]))
assert.eq(text, """i64_rep: 1
i64_rep: 2
i64_rep: 3
""")
