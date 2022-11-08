from py2x3 import *


def ExportFromCatalog(v_lo, v_hi, from_catalog = None):
    if from_catalog == None:
        from_catalog = GetPrimeCatalog(v_hi)
    
    sel = NewSelector()
    sel.min.verts = v_lo
    sel.max.verts = v_hi
    
    dst = "learn/gold/"
    
    sel.primes = False
    sel.unique_traces = False
    from_catalog.Select(sel).Print("Complete",  traces=8, file=dst+"2x3 Complete Catalog.csv"       ).Go()
    sel.primes = True
    sel.unique_traces = True
    from_catalog.Select(sel).Print("Primes",    traces=8, file=dst+"2x3 Primes Catalog.csv"         ).Go()
    
    sel.max.arrows = 0
    sel.primes = False
    sel.unique_traces = False
    from_catalog.Select(sel).Print("Pure",      traces=8, file=dst+"2x3 Pure Catalog.csv"           ).Go()


ExportFromCatalog(1,6)

