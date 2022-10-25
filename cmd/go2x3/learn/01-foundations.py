print('''
Welcome to the 2x3 Workspace tutorial series!
2x3 Workspace facilitates calculations for 2x3 Particle Theory =.''')

# First we must import the 2x3 module.  Consider opening and exploring lib/py2x3.py
from py2x3 import *

# To express particle graphs, we make a new Graph from a vertex and edge initialization string. 
# Vertices are labeled as positive integers and single (positive) edges appear as a dash.
#
# Recall each vertex in 2x3 has a combined total of 3 loops and/or edges.
# Vertices that have less than three edges are assumed to have positive self-connected edges (loops).

# A proton has two loops on each end ond one in the middle
proton = NewGraph("1-2-3")

# An electron is a single vertex with three loops
electron = NewGraph("1")

# We can output a Graph object to the console by using the str() conversion operator.  
# This implicitly happens for ops like print() that auto-convert into a string.
print("Our friend the proton: ", proton)

# Vertices that are double or triple connected use multiple dashes. 
photon = NewGraph("1---2")

# When a caret ('^') appears after a vertex, it means to replace a loop with an arrow.
positron = NewGraph("1^^^")

# A Z0 boson has two vertices, each with a loop and an arrow.
# Also, for convenience, using "Graph()" is equivalent to "NewGraph()""
z_boson = Graph("1^-2^")

# For graphs that can't be expressed in a single edge "run", use commas to serparate multiple runs
higgs = Graph("1-2-3-4-1-5-6-7-8-5, 2-6, 3-7, 4-8")


print('''
If we want to get the traces of a 2x3 graph, we can!''')
T1 = proton.Traces()
print("proton.Traces(): ", T1)

# Or, we can ask for more Trace elements to be computed.
T2 = proton.Traces(10)
print("proton.Traces(10): ", T2)


print('''
Or we can access each Traces element easily thanks to python:''')
for i in range(len(T2)):
    print("    T[%d]: %d" % (i, T2[i]))


print()

print('''Now let's meet GraphStream, allowing us to do more useful things with particle Graph objects.
GraphStream is a chain (pipeline) of Graph operators where Graphs are "pushed" into a stream's inlet and "pulled" from its outlet. 
Adding .Print(label) prints each graph that passes though the stream, printing the given label along with a counter.
Adding .Go() terminates a stream and pulls all the graphs through it:''')
proton.Stream().Print("Hello proton!").Go()


# Some commands "fan out", meaning for each given graph, they output multiple graphs
# Two of these are AllVtxSigns() and AllEdgeSigns()...
# AllVtxSigns() emits all possible loops & arrows permutations for each input particle.
print('''
For each particle that AllVtxSigns() takes in, it sends out all possible permutations of loops and arrows:''')
electron.AllVtxSigns().Print("electron.AllVtxSigns").Go()

print('''
The more loops and arrows that are present, the more permutations that are possible:''')
proton.AllVtxSigns().Print("proton.AllVtxSigns").Go()

print('''
For particles containing only 'gamma' vertices, only one permutation is possible:''')
photon.AllVtxSigns().Print("gamma.AllVtxSigns").Go()

print('''
All edge permutations for a two-vertex photon:''')
photon.AllEdgeSigns().Print("gamma.AllEdgeSigns").Go()

# AllEdgeSigns() emits all possible edge sign permutations for each input particle.
print('''
For each particle that AllEdgeSigns() takes in, it sends out all possible permutations of positive and negative edges:''')
proton.AllEdgeSigns().Print("proton.AllEdgeSigns").Go()


# Since we're using python, we can use data structures like lists and dicts to make life easier:
neutrons = {}

print('''
Most graphs can be expressed identically in multiple ways.
Consider various equivalent ways to label a neutron:''')
neutrons["a"] = Graph("1-2-3-1-4")
neutrons["b"] = Graph("1-2-3-1, 2-4")
neutrons["c"] = Graph("3-4,2-3-1-2")
print("neutron[a]: ", neutrons["a"])
print("neutron[b]: ", neutrons["b"])
print("neutron[c]: ", neutrons["c"])


print('''
EnumPureParticles() runs an algorithm that generates all possible valid particles having only loops and positive edges.
This algorithm is mechanical and will therefore repeat equivalent particles having different labels. 
Let's generate from v=1 to v=3:''')
EnumPureParticles(1,3).Print().Go()


print('''
The DropDupes() operator only emits unique graphs while dropping duplicates.''')
EnumPureParticles(1,3).DropDupes().Print().Go()

# Now we introduce a powerful GraphStream() operator called Select() that is a filter and selects Graphs that meet criteria that we provide.
# The Select() operator uses a struct called a GraphSelector that specifies which graphs you want to select.
sel = NewSelector()

print('''
We can select by matching Traces:''')
sel.traces = photon
EnumPureParticles(1,3).Select(sel).Print().Go()

print('''
We can also select by min and max vertex count:''')
sel.traces = None
sel.min.verts = 1
sel.max.verts = 2
EnumPureParticles(1,3).Select(sel).Print().Go()

print('''
We can select by loops (or arrows):''')
sel.Init()
sel.min.loops = 2
sel.max.loops = 3
EnumPureParticles(1,3).Select(sel).Print().Go()

print('''
We can select by loops AND arrows:''')
sel.Init()
sel.min.loops = 2
sel.max.loops = 3
sel.min.arrows = 1
sel.max.arrows = 100
proton.AllVtxSigns().Select(sel).Print().Go()



