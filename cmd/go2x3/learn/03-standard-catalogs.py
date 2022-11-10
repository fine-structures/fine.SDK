from py2x3 import *


def ExportCatalogsToCSV(v_lo, v_hi, from_catalog = None):
    if from_catalog == None:
        from_catalog = GetPrimeCatalog(v_hi)
    
    sel = NewSelector()
    sel.min.verts = v_lo
    sel.max.verts = v_hi
    
    dst = "Standard Catalogs/"
    
    sel.primes        = False
    sel.unique_traces = False
    from_catalog.Select(sel).Print("Extended",  traces=8, file=dst+"Extended Catalog.csv"   ).Go()
    
    sel.primes        = False
    sel.unique_traces = True
    from_catalog.Select(sel).Print("Complete",  traces=8, file=dst+"Complete Catalog.csv"   ).Go()
    
    sel.primes        = True
    sel.unique_traces = True
    from_catalog.Select(sel).Print("Primes",    traces=8, file=dst+"Prime Catalog.csv"      ).Go()
    
    sel.max.neg_loops = 0
    sel.max.neg_edges = 0
    sel.primes        = False
    sel.unique_traces = False
    from_catalog.Select(sel).Print("Pure",      traces=8, file=dst+"Pure Catalog.csv"       ).Go()

    sel.max.pos_loops = 0
    sel.primes        = False
    sel.unique_traces = False
    from_catalog.Select(sel).Print("Boson",     traces=8, file=dst+"Boson Catalog.csv"      ).Go()


def PrintTricodeCatalog(v_hi, from_catalog = None):
    if from_catalog == None:
        from_catalog = GetPrimeCatalog(v_hi)
        
    sel = NewSelector()
    sel.min.verts = 1
    sel.max.verts = v_hi
    
    sel.primes = False
    sel.unique_traces = True
    from_catalog.Select(sel).Print("Complete", codes=True).Go()
        

ExportCatalogsToCSV(1,6)

#PrintTricodeCatalog(4)




