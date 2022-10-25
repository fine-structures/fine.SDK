# go2x3

### Official Go toolkit for [2x3 Particle Theory](https://github.com/2x3systems/prime-materials), a [Standard Model](https://en.wikipedia.org/wiki/Standard_Model) consolidation theory 

## Quick Start

First, with [Go](https://go.dev/doc/install) installed, build the `go2x3` binary:
```bash
% make build
% cd cmd/go2x3 && ls learn
```

Then explore or run any of the tutorial scripts:
```bash
% ./go2x3 learn/01-foundations.py
% ./go2x3 learn/02-techniques.py
% ./go2x3 learn/03-standard-catalogs.py
% ./go2x3 learn/04-neutron-decay.py
% ./go2x3 learn/05-cycles-deep-dive.py
% ./go2x3 learn/06-lepton-non-universality.py
```

You can also skip the above and go directly to the [learn](https://github.com/2x3systems/go2x3/tree/main/cmd/go2x3/learn) scripts and their "[gold](https://github.com/2x3systems/go2x3/tree/main/cmd/go2x3/learn/gold)" output.


## Upcoming
- Reimplement `GraphWalker` so that graph construction steps map to deterministic vertex and edge placement (graph visualization)
- Replace particle `Traces` catalog indexing with a `TriID` characteristic extraction operator


## Releases

| Version   | Description                                                                               |
|:---------:|:-------------------------------------------------------------------------------------------------|
| v1.2022.1 | - traces-based particle catalog index  <br/> - conventional (non-canonic) vertex-based graph encoding  <br/> - introducing early tricodes   |
