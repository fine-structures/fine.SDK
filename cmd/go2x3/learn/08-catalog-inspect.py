from py2x3 import *

print("This outputs every isomorphically unique graph up to v=4\n\n")

# Auto generate catalog up to at least 3 vertices
catalog = GetPrimeCatalog(4)

printOpts = {
    'graph':  True,
    'traces': 12,
    'cycles': True,
}


sel = NewSelector()
sel.max.verts = 6
sel.unique_traces = True
sel.primes = False
catalog.Select(sel).Print("ALL", **printOpts).Go()

