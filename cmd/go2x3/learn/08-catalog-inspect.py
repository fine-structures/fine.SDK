from py2x3 import *

print("This outputs every isomorphically unique graph up to v=4\n\n")

# Auto generate catalog up to at least 3 vertices
catalog = GetPrimeCatalog(4)

printOpts = {
    'graph':  True,
    'codes':  True,
    'traces': 10,
}


sel = NewSelector()
sel.max.verts = 4
sel.unique_traces = False
sel.primes = False
catalog.Select(sel).Print("ALL", **printOpts).Go()
