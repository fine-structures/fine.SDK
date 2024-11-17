from py2x3 import *

# Auto generate catalog up to at least 8 vertices
catalog = GetPrimeCatalog(8)

dst = "Standard Catalogs/"

sel = NewSelector()
sel.max.verts = 6
sel.unique_traces = False
catalog.Select(sel).Print("Extended",   traces=12, file=dst+"Extended Catalog.csv"   ).Go()
    
sel.unique_traces = True
catalog.Select(sel).Print("Complete",   traces=12, file=dst+"Complete Catalog.csv"   ).Go()
    
sel.unique_traces = True
sel.select_primes = True
catalog.Select(sel).Print("Primes",     traces=12, file=dst+"Prime Catalog.csv"      ).Go()
    

sel = NewSelector()
sel.max.verts = 8
sel.max.neg_loops = 0
sel.max.neg_edges = 0
sel.unique_traces = False
catalog.Select(sel).Print("Pure",       traces=12, file=dst+"Pure Catalog.csv"       ).Go()

sel = NewSelector()
sel.max.verts = 8
sel.unique_traces = True
sel.select_bosons = True
catalog.Select(sel).Print("Boson",      traces=12, file=dst+"Boson Catalog.csv"      ).Go()

