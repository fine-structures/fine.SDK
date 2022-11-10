
# First we must import the 2x3 module.  Consider opening and exploring lib/py2x3.py
from py2x3 import *


print('''
A Catalog is a collection (a database) of Graphs.  
When creating a Catalog, if no filename is specified, a memory-resident Catalog is created instead.
To add particles to a Catalog, use the GraphStream operator .AddTo():''')
catA = NewCatalog()
EnumPureParticles(1,1).AddTo(catA).Print("adding v=1").Go()
EnumPureParticles(3,3).AddTo(catA).Print("adding v=3").Go()

print('''
When we call Select() from a catalog without any selection criteria, we'll get the whole catalog back:''')
catA.Select().Print("All").Go()

print('''
If we try to add particles that are already present in a catalog, they are dropped.
Notice that particles of v=1 and 3 do not make it past .AddTo(), allowing you see only particles that have been added.''')
EnumPureParticles(1,3).AddTo(catA).Print("adding v=1,2,3").Go()

print('''
Let's just select particles with 3 vertices and at least 2 loops:''')
sel = NewSelector()
sel.min.verts = 3
sel.max.verts = 3
sel.min.loops = 3
catA.Select(sel).Print("v=3 & loops > 2").Go()


print('''
So far, our catalog is full of equivalent particles with different labeling.
AddTo() will properly drop all dupes:''')
catB = NewCatalog()
catA.Select().AddTo(catB).Print("no dupes").Go()
catB.Close()

print('''
It's common to filter duplicate Graphs, so a convenience function DropDupes() does the same as the above:''') 
catA.Select().DropDupes().Print("still no dupes").Go()


print('''
Let's put everything we've learned together and create a complete catalog of all possible 2x3 particles.
First, we generate a catalog of all possible particles containing while keeping all edges (and self-edges) to be positive.
We call this the "pure matter" catalog since particles with only loops and positive edge correspond to matter in positive time and space.
Pure matter includes electrons, muons, protons, neutrons, and neutrinos.''')
v_hi = 3
pure_matter = NewCatalog()
count = EnumPureParticles(1,v_hi).AddTo(pure_matter).Print("Pure").Go()
print("...so there's exactly *%d* particles in the pure matter catalog for v=1..%d, nice!" % (count, v_hi))

print('''
How about a catalog that includes anti-matter particles?  For that, we use AllVtxSigns().
We call this the "mixed matter" catalog since contains it particles constructions with negative loops (negative self-edges).''')
mixed_matter = NewCatalog()
pure_matter.Select().AllVtxSigns().AddTo(mixed_matter).Print("Mixed").Go()

print('''
Finally, let's include *all* possible particles using AllEdgeSigns().
We call this the "complete" catalog since every possible particle (having any combination of negative edges).''')
complete_catalog = NewCatalog()
mixed_matter.Select().AllEdgeSigns().AddTo(complete_catalog).Print("Complete", file="learn/gold/My First Catalog.csv", matrix=True).Go()

print('''
With a complete catalog, we can query for a given particle's Traces and get a list of all possible equivalent "phase modes":''')
muon = Graph("1-2--3")
sel = NewSelector()
sel.traces = muon
complete_catalog.Select(sel).DropDupes().Print("muon phases (long way)").Go()      

print('''
The function PhaseModes() wraps the above into a convenient function.
Check out lib/lib2x3.py for other utility functions.''')
muon.PhaseModes(complete_catalog).Print("muon phases (PhaseModes)").Go()


