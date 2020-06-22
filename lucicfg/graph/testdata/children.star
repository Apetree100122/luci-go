def test_children_query():
    g = new_graph()

    def mk(*args):
        k = g.key(*args)
        g.add_node(k)
        return k

    par = mk("parent", "parent")

    def q(parent, order_by):
        keys = []
        for n in g.children(parent, order_by):
            keys.append(n.key)
        return keys

    t1_a1 = mk("t1", "id1", "a", "1")
    t1_a2 = mk("t1", "id1", "a", "2")
    t1_b1 = mk("t1", "id1", "b", "1")
    t2_a1 = mk("t2", "id1", "a", "1")

    g.add_edge(par, t1_b1)
    g.add_edge(par, t2_a1)
    g.add_edge(par, t2_a1, title = "another rel")
    g.add_edge(par, t1_a2)
    g.add_edge(par, t1_a1)

    g.finalize()

    # Querying in 'def' order returns nodes in order we defined edges.
    assert.eq(q(par, "def"), [t1_b1, t2_a1, t1_a2, t1_a1])

    # Querying by key order returns nodes in lexicographical order.
    assert.eq(q(par, "key"), [t1_a1, t1_a2, t1_b1, t2_a1])

    # Asking for children of non-existing node is OK.
    assert.eq(q(g.key("t", "missing"), "key"), [])

    # order_by is validated.
    assert.fails(lambda: q(par, "blah"), 'unknown order "blah"')

test_children_query()
