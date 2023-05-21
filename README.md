# _go2x3_
### Original SDK for [2x3 Particle Theory](https://github.com/2x3systems/prime-materials), a [Standard Model](https://en.wikipedia.org/wiki/Standard_Model) unifying theory.

------------------------------



## Quick Start

With [Go](https://go.dev/doc/install) installed, build the `go2x3` binary:
```bash
% make build
% cd cmd/go2x3 && ls learn
```

Explore or run any of the tutorial scripts:
```bash
% ./go2x3 learn/00-hello-electron.py
% ./go2x3 learn/01-foundations.py
% ./go2x3 learn/02-techniques.py
% ./go2x3 learn/03-standard-catalogs.py
% ./go2x3 learn/04-neutron-decay.py
% ./go2x3 learn/05-cycles-deep-dive.py
% ./go2x3 learn/06-lepton-non-universality.py
% ./go2x3 learn/07-factors-of-light.py
% ./go2x3 learn/08-catalog-inspect.py
```

Or skip the above and go directly to the [learn](https://github.com/2x3systems/go2x3/tree/main/cmd/go2x3/learn) scripts and their "[gold](https://github.com/2x3systems/go2x3/tree/main/cmd/go2x3/learn/gold)" output.

## Getting Started

This project is a library & demonstration toolkit for [2x3 Particle Theory](https://github.com/2x3systems/prime-materials).  Although [lib2x3](http://https://github.com/2x3systems/go2x3/tree/main/lib2x3) is a pure Go library, [`gpython`](http://github.com/go-python/gpython) is used to embed and expose it.  This means scripting is easy  — see for yourself in the [first tutorial](https://github.com/2x3systems/go2x3/blob/main/cmd/go2x3/learn/01-foundations.py) as you follow along in its [output](https://github.com/2x3systems/go2x3/blob/main/cmd/go2x3/learn/gold/01-foundations.txt).


## Releases

| Version   | Description                                                                               |
|:---------:|:-------------------------------------------------------------------------------------------------|
| v1.2022.1 | - traces-based particle catalog index  <br/> - conventional (non-canonic) vertex-based graph encoding  <br/> - introducing early tricodes   |
| v1.2022.2 | - refactor graph canonicalization  <br/> - refactor tricode console output |
| v1.2022.3 | - graph ascii art <br/> - refactor tricode encoding |
| v1.2023.1 | - refactor and cleanup |
| v1.2023.2 | - constituent edge traces factorization (WIP) |

