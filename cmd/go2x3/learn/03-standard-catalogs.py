from py2x3 import *


def ExportFromCatalog(v_lo, v_hi, from_catalog = None):
    if from_catalog == None:
        from_catalog = GetPrimeCatalog(v_hi)
    
    sel = NewSelector()
    sel.min.verts = v_lo
    sel.max.verts = v_hi
    
    dst = "Standard Catalogs/"
    
    sel.primes = False
    sel.unique_traces = False
    from_catalog.Select(sel).Print("Complete",       traces=8, file=dst+"Complete Catalog.csv"       ).Go()
    sel.primes = True
    sel.unique_traces = True
    from_catalog.Select(sel).Print("Complete Prime", traces=8, file=dst+"Complete Primes.csv"        ).Go()

    sel.max.neg_edges = 0
    sel.primes = False
    sel.unique_traces = False
    from_catalog.Select(sel).Print("Mixed",          traces=8, file=dst+"Mixed Matter Catalog.csv"   ).Go()
    sel.primes = True
    sel.unique_traces = True
    from_catalog.Select(sel).Print("Mixed Prime",    traces=8, file=dst+"Mixed Matter Primes.csv"    ).Go()
    
    sel.max.arrows = 0
    sel.primes = False
    sel.unique_traces = False
    from_catalog.Select(sel).Print("Pure",           traces=8, file=dst+"Pure Matter Catalog.csv"    ).Go()
    sel.primes = True
    sel.unique_traces = True
    from_catalog.Select(sel).Print("Pure Prime",     traces=8, file=dst+"Pure Matter Primes.csv"     ).Go()



ExportFromCatalog(1,6)

