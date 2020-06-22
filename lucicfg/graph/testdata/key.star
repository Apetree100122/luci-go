def test_keys_work():
    g = new_graph()

    k1 = g.key("t1", "id1")
    assert.eq(type(k1), "graph.key")
    assert.eq(str(k1), '[t1("id1")]')
    assert.true(k1)

    k2 = g.key("t1", "id1")  # exact same as k1
    assert.true(k1 == k2)

    k3 = g.key("t1", "id1", "t2", "id2")
    assert.true(k1 != k3)
    assert.eq(str(k3), '[t1("id1"), t2("id2")]')

    # Can be used as key in dicts.
    d = {}
    d[k1] = "1"
    assert.eq(d[k2], "1")

    # '@' in front is OK.
    g.key("@t1", "id")
    g.key("@t1", "id", "t2", "id")

    # '@' in IDs is not syntactically significant.
    g.key("t1", "@id")

def test_keys_attrs():
    g = new_graph()

    k = g.key("k1", "i1", "k2", "i2", "k3", "i3")
    assert.eq(dir(k), ["container", "id", "kind", "root"])
    assert.eq(k.id, "i3")
    assert.eq(k.kind, "k3")

    assert.eq(k.container, g.key("k1", "i1", "k2", "i2"))
    assert.eq(k.container.container, g.key("k1", "i1"))
    assert.eq(k.container.container.container, None)

    assert.eq(k.root, g.key("k1", "i1"))
    assert.eq(k.root.root, k.root)  # an invariant for roots

def test_keys_fail():
    g = new_graph()

    assert.fails(lambda: g.key(), "empty key path")
    assert.fails(lambda: g.key("t1"), "has odd number of components")
    assert.fails(
        lambda: g.key("t1", None),
        "all arguments must be strings, arg #1 was NoneType",
    )
    assert.fails(lambda: g.key("t1", "i1\000sneaky"), "has zero byte inside")

    assert.fails(
        lambda: g.key("t1", "id", "@t1", "id"),
        'kind "@t1" can appear only at the start of the key path',
    )

def test_keys_incomparable():
    # Keys from different graphs are not equal, even if they have same path.
    k1 = new_graph().key("t1", "id1")
    k2 = new_graph().key("t1", "id1")
    assert.true(k1 != k2)

test_keys_work()
test_keys_attrs()
test_keys_fail()
test_keys_incomparable()
