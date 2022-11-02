import _py2x3

# Global variables
MAX_VTX = _py2x3.MAX_VTX

'''
|    WHERE WERE YOU WHEN I LAID THE FOUNDATION OF THE EARTH?    |
|               TELL ME, IF YOU HAVE UNDERSTANDING.             |
|               TELL ME, IF YOU KNOW ALL THIS.                  |
'''
print('''
=================================================================
|                       go2x3 (%s)                       |
|                      https://2x3.systems                      |
|                                                               |
|       "OH LET ME BEHOLD YOUR PRESENCE”  EXODUS 33:2x3x3       |
=================================================================
''' % (_py2x3.LIB2x3_VERSION))

V_nil    = int(0)
V_e      = int(1)
V_e_bar  = int(2)
V_π      = int(3)
V_pi     = int(3)
V_π_bar  = int(4)
V_pi_bar = int(4)
V_u      = int(5)
V_u_bar  = int(6)
V_q      = int(7)
V_d      = int(8)
V_d_bar  = int(9)
V_𝛾      = int(10)
V_y      = int(10)


def EnumPureParticles(v_lo, v_hi):
    return _py2x3.EnumPureParticles(v_lo, v_hi)

def NewGraph(*parts):
    return Graph(*parts)

class Graph:

    def __init__(self, *parts):
        self._graph = _py2x3.NewGraph()
        self.Concat(*parts)
        
    def __str__(self):
        return str(self._graph)

    def NumVerts(self):
        return self._graph.NumVerts()
        
    def NumParts(self):
        return self._graph.NumParts()
        
    def Traces(self, num_traces = 0):
        return self._graph.Traces(num_traces)

    def Concat(self, *parts):
        self._graph.Concat(parts)

    def Stream(self):
        return self._graph.Stream()

    def Print(self, *args, **kwargs):
        """Prints each Graph from a GraphStream with various options

        Available kwargs: 
            label = string          - Sets a label for this output run 
            traces = int            - Prints the graph's first N Traces (N=0 denotes the vertex count)
            cycles = bool           - Prints cycle computation details
            uid = bool              - Prints the graph's canonic UID 
            file = <pathname>       - Echos output to the given file pathname 
        """
        return self._graph.Stream().Print(*args, **kwargs)

    def AddTo(self, catalog):
        return self._graph.Stream().AddTo(catalog)
        
    def Select(self, graphSelector):
        return self._graph.Stream().Select(graphSelector)
        
    '''
    Emits all canonically unique permutations of given graph's edge signs.
    '''
    def EdgeModes(self):
        sel = GraphSelector()
        sel.traces = self
        return self.AllEdgeSigns().Select(sel).DropDupes()

    '''
    Emits all canonically unique permutations of given graph's edge signs.
    '''
    def PhaseModes(self, from_catalog = None):
        if from_catalog == None:
            from_catalog = GetPrimeCatalog(self.NumVerts())
            
        sel = GraphSelector()
        sel.traces = self
        return from_catalog.Select(sel)
        
    '''
    Emits all canonically unique *prime* particle combinations for this Graph (having equal Traces).
    '''
    def PrimeModes(self, prime_catalog = None):
        if prime_catalog == None:
            prime_catalog = GetPrimeCatalog(self.NumVerts())
            
        sel = GraphSelector()
        sel.traces = self
        sel.factor = True
        return prime_catalog.Select(sel)
        
    '''
    Emits all canonically unique particle combinations for this Graph (having equal Traces).
    '''
    # def DecayModes(self, prime_catalog = None):
    #     if prime_catalog == None:
    #         prime_catalog = GetPrimeCatalog(self.NumVerts())
                    
    #     cat.Close()
    #     pass

    def AllVtxSigns(self):
        return self._graph.Stream().AllVtxSigns()
        
    def AllEdgeSigns(self):
        return self._graph.Stream().AllEdgeSigns()
        
        
        
class GraphInfo:
    """
    GraphInfo is a simple struct that specifies info about a 2x3 graph.  GraphInfo is used in GraphSelector to select min and max stats for graphs to be selected.
    """

    def __init__(self):
        self.parts = 0
        self.verts = 0
        self.pos_edges = 0
        self.neg_edges = 0
        self.loops = 0
        self.arrows = 0


def NewSelector(*parts):
    return GraphSelector(traces = None)
    
def Select(sel):
    GetPrimeCatalog().Select(sel)

class GraphSelector:
    """
    GraphSelector is passed to <GraphStream>.Select() to specify one or more restricting parameters that allow what graphs are selected and which are filtered (blocked).
    """

    def __init__(self, traces = None):
        self.min = GraphInfo()
        self.max = GraphInfo()
        self.Init()
        self.traces = traces

    def Init(self):
        self.traces = None
        self.factor = False
        self.primes = False
        self.unique_traces = False

        self.min.parts = 1
        self.min.verts = 1
        self.min.pos_edges = 0
        self.min.neg_edges = 0
        self.min.loops  = 0
        self.min.arrows = 0

        self.max.parts = MAX_VTX
        self.max.verts = MAX_VTX
        self.max.pos_edges = int(MAX_VTX*3/2)
        self.max.neg_edges = int(MAX_VTX*3/2)
        self.max.loops  = MAX_VTX*3
        self.max.arrows = MAX_VTX*3
        
    def SetTraces(self, X):
        self.traces = X.Traces()
    


READ_ONLY           = 0x01
READ_WRITE          = 0x02
PRIME_CATALOG       = 0x04

def NewCatalog(pathname = "", flags = READ_WRITE):
    return Catalog(pathname, flags)
    
class Catalog:
    """
    A Catalog is a store for particles and can be mapped to the heap (for short-term use) or to a file (for persistency).
    """
    def __init__(self, pathname = "", flags = READ_ONLY, minTraceCount = 0):
        self._isReadOnly = (flags & READ_ONLY) != 0
        if len(pathname) == 0:
            if self._isReadOnly:
                flags = (flags ^ READ_ONLY)
        self.default_selector = GraphSelector()
        self._cat = _py2x3.GetWorkspace().OpenCatalog(pathname, flags, minTraceCount)
        
    def Select(self, graph_selector = None):

        # If no selector is given, use this Catalog's default GraphSelector
        if graph_selector == None:
            graph_selector = self.default_selector

        # returns a GraphStream
        return self._cat.Select(graph_selector)
        
    def NumTraces(self, forVtxCount):
        return self._cat.NumTraces(forVtxCount)
    
    def NumPrimes(self, forVtxCount):
        return self._cat.NumPrimes(forVtxCount)

    def Close(self):
        if self._cat != None:
            self._cat.Close()
            self._cat = None
            
        
    def CalcPrimes(self, toVtxCount):
    
        # See if the existing catalog has primes generated to the needed level.
        # If not, we must re-open the catalog for writing.
        if self.NumPrimes(toVtxCount) > 0:
            return
            
        print("########################################################################")
        print("##        On-Demand Prime Particle Catalog Generation to v=%-2d         ##" % toVtxCount)
        print("##____________________________________________________________________##")

        total_count = 0
        vi = 1
        while vi <= toVtxCount:
            
            # Skip vtx levels until we find the end of what's already been calculated
            if self.NumPrimes(vi) == 0:
             
                count = EnumPureParticles(vi, vi)       \
                    .DropDupes().AllVtxSigns()          \
                    .DropDupes().AllEdgeSigns()         \
                    .AddTo(self).Go()
        
                print("##  v=%2d:%13d graphs %11d traces %11d primes   ##" % 
                            (vi, count, self.NumTraces(vi), self.NumPrimes(vi)))
        
                total_count += count
                
            vi += 1

        print("########################################################################")


_DefaultPrimeCatalog = None

def GetPrimeCatalog(toVtxCount = 8):
    global _DefaultPrimeCatalog
    
    default_name = "catalogs/_DefaultPrimeCatalog"

    # Almost all of the time, we won't need to calculate primes (only the first time)
    if _DefaultPrimeCatalog == None and _py2x3.GetWorkspace().CatalogExists(default_name):
        _DefaultPrimeCatalog = Catalog(default_name, READ_ONLY | PRIME_CATALOG)
            
    # See if the existing catalog has primes generated to the needed level.
    # If not, we must re-open the catalog for writing.
    if _DefaultPrimeCatalog == None or _DefaultPrimeCatalog.NumPrimes(toVtxCount) == 0:
        if _DefaultPrimeCatalog != None:
            _DefaultPrimeCatalog.Close()

        cat = Catalog(default_name, READ_WRITE | PRIME_CATALOG)
        cat.CalcPrimes(toVtxCount)
        cat.Close()
        
        _DefaultPrimeCatalog = Catalog(default_name, READ_ONLY | PRIME_CATALOG)

    return _DefaultPrimeCatalog
