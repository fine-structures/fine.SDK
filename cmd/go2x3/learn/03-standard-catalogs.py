from py2x3 import *

# Auto generate catalog up to at least 8 vertices
catalog = GetPrimeCatalog(8)

dst = "Standard Catalogs/"

sel = NewSelector()
sel.max.verts = 6
sel.unique_traces = False
sel.primes = False
catalog.Select(sel).Print("Extended",   traces=8, file=dst+"Extended Catalog.csv"   ).Go()
    
sel.unique_traces = True
sel.primes = False
catalog.Select(sel).Print("Complete",   traces=8, file=dst+"Complete Catalog.csv"   ).Go()
    
sel.unique_traces = True
sel.primes = True
catalog.Select(sel).Print("Primes",     traces=8, file=dst+"Prime Catalog.csv"      ).Go()
    

sel = NewSelector()
sel.max.verts = 8
sel.max.neg_loops = 0
sel.max.neg_edges = 0
sel.unique_traces = False
sel.primes = False
catalog.Select(sel).Print("Pure",       traces=8, file=dst+"Pure Catalog.csv"       ).Go()
    

sel.max.pos_loops = 0
catalog.Select(sel).Print("Boson",      traces=8, file=dst+"Boson Catalog.csv"      ).Go()

