# Guarded deepmerge

This carries the MIT-licensed bundled CommonJS implementation and declarations from deepmerge 4.3.1. The upstream licence is retained in LICENSE.

GHSA-gcqx-w48m-w2h4 describes prototype poisoning of returned objects when merging keys from null-prototype or specially constructed inputs. No patched upstream version was available on 28 October 2026.

The fork filters `__proto__`, `constructor`, and `prototype` from both inputs before reading their values and defines copied keys as own data properties. Throwing membership checks propagate their error. Arrays, symbols, custom merge functions, and the upstream clone option retain their normal behaviour. `clone: false` and caller-supplied merge functions still own the safety of objects they return or retain.

`npm run test:dependencies` exercises the installed transitive package and adversarial inputs. Remove this fork and its override when an upstream release passes those regression tests. The local package name and version identify the maintained code; they do not claim an upstream fix.

Advisory: https://github.com/advisories/GHSA-gcqx-w48m-w2h4
Upstream: https://github.com/TehShrike/deepmerge/tree/v4.3.1
