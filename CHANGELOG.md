# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### <!-- 0 -->🚀 Features

- add sb.vookie.net to forward_auth registry ([649c6cc](649c6cca39b65ccf12737fba986d0594aeb9cc09))

- add forward_auth registry to detect missing auth regressions ([66c9df2](66c9df25888e3295a1f98d7c3e97fda20e51064f))


### <!-- 1 -->🐛 Bug Fixes

- fix trailing whitespace in built/static files, changelog push perms ([553f9f6](553f9f61976e8795af2b155d5c72a58791b03fd9))

- fix pre-commit formatting issues and changelog workflow ([65582b2](65582b29e8453cac7d445fd01c24570050cf04c8))

- fix cliff.toml for git-cliff v4 compatibility ([15b4839](15b4839352c866ca17e3243b6f5728d76928f6f5))

- fix shellcheck word splitting, upgrade git-cliff-action ([28d27d4](28d27d4eac820689369ce4d0d5a513929fb300f5))

- make trivy non-blocking, fix npm ci peer dep conflict in build/lint ([1435dba](1435dba553f475971f3bc9d2261ec924775aa029))

- replace broken trivy-action, fix govulncheck and npm ci ([710cd11](710cd11683d9964983079a50659d172f58dc226f))

- concurrency control, security hardening, and reliability improvements ([c184cb9](c184cb9eda8652f2fdd2c8ea76f270348ce1007d))


### <!-- 7 -->⚙️ Miscellaneous Tasks

- fix branch refs, add badges, CodeRabbit, fix Build/Lint ([f63ef03](f63ef03a8cf7d5d03dcb19015bd5fc6361b4a963))

- replace Dependabot with Renovate ([fe5fad7](fe5fad747ec4efca166506164a08d13a0d238541))

- add CVE scanning, Dependabot, SECURITY.md, enable gosec ([2bb3287](2bb3287d7e8aa81249b0aa9a23506e6701855dd1))


## [0.4.54] - 2026-07-29

### <!-- 0 -->🚀 Features

- graceful shutdown, SSE diagnostics stream, repair-dns progress ([0c98416](0c98416c7df80b747677149fd4ddd14b6f3047f3))


### <!-- 4 -->⚡ Performance

- add context propagation, panic recovery, and entries caching ([2ab1889](2ab1889d789982f383b016da58734217fbee4976))


## [0.4.53] - 2026-07-29

### <!-- 1 -->🐛 Bug Fixes

- clean up error handling, validation, and minor code quality issues ([9b67f16](9b67f16d2941c87dc0d32c657773a329fbff8d46))


## [0.4.52] - 2026-07-29

### <!-- 2 -->🚜 Refactor

- improve error handling, validation, and code organization ([5a2cad7](5a2cad7f9f9704826e18b99d0ab57d10949dcf1e))


## [0.4.51] - 2026-07-29

### <!-- 0 -->🚀 Features

- auto-detect Authentik IdP host to avoid false positive ([6e51aad](6e51aad71bbb3a44e9fcae82cd82d48a82ef4ed8))


### <!-- 1 -->🐛 Bug Fixes

- critical build, CI, and code quality issues ([1bad4de](1bad4deeb866c774dd87d0d4a528a74c1288def9))

- flag WAN-exposed hosts with CF Access bypass-only as critical error ([e571e6c](e571e6cac415f8d7198038dac5989e31f2c49b58))


### <!-- 2 -->🚜 Refactor

- split monolithic server.go into domain-specific files ([78d0563](78d0563cf5b53262b2933585b4dd6e940648a167))

- deprecate conditional forward_auth (Pattern E) ([3e59df0](3e59df0cb32b6ec16b8d7ee7ca646d380ba6ec41))


## [0.4.50] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- detect conditional forward_auth to avoid false double-login warning ([eba90a6](eba90a6c507683954be2ef0053160b5f87007da9))


## [0.4.49] - 2026-07-28

### <!-- 0 -->🚀 Features

- standalone visualize page at /visualize/{hostname} ([5ffe119](5ffe1193c0bfa784f681f10b170b4b97aca6b9f6))


## [0.4.48] - 2026-07-28

### <!-- 0 -->🚀 Features

- deep link support for visualize modal ([11298d9](11298d92b7337956908c563503a9592e638a0338))


## [0.4.47] - 2026-07-28

### <!-- 10 -->💼 Other

- replace bottom tables with pill badges ([2002c71](2002c71fe2d15ace1ecb9dc33b2cfb52fae5278d))


## [0.4.46] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- increase ELK node spacing to reduce cramped layout ([173e236](173e236d98e8ed17b5aeb2531ab3c7ee88da1a85))


## [0.4.45] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- always use vertical layout for consistency ([6ea64cd](6ea64cdef4bc4f71f3eef15756fac3ccbe069db0))


## [0.4.44] - 2026-07-28

### <!-- 0 -->🚀 Features

- show service name instead of 'Service' in diagram nodes ([7745de5](7745de562d4168367bf0db75af419d3f44dbae49))


## [0.4.43] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- arrows now visible, move IdP label to JWT verified ([4a26a59](4a26a59aff16b47a883c55ae1986fea89529005a))


## [0.4.42] - 2026-07-28

### <!-- 5 -->🎨 Styling

- white circle badges with blue outline and blue number ([d536ad0](d536ad0a73f016744602ec05e83421f67354a09e))


## [0.4.41] - 2026-07-28

### <!-- 2 -->🚜 Refactor

- replace flow table with vertical step list ([d83cbb0](d83cbb05df4a8164ab6da5e7f166604c959a7a4c))


## [0.4.40] - 2026-07-28

### <!-- 0 -->🚀 Features

- side-by-side diagram+flow layout, custom arrow edges, step numbers ([2d55515](2d555158278b80d879b104ea22c420270d0b25fb))


## [0.4.39] - 2026-07-28

### <!-- 5 -->🎨 Styling

- widen VisualizeModal from 720px to 1100px ([50e2a4b](50e2a4b7cf8da06ef7ce7391f1a0300cf5857ce2))


## [0.4.38] - 2026-07-28

### <!-- 2 -->🚜 Refactor

- group WAN diagram+flow together, LAN diagram+flow together ([9a33005](9a33005d5b0c6d14592b03e092dd30d9bb898d31))


## [0.4.37] - 2026-07-28

### <!-- 2 -->🚜 Refactor

- replace status tile grids with config tables ([dfaef19](dfaef192bb3d17c524c679857430ecbc29f42933))


## [0.4.36] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- React Flow diagrams — fitView after layout, vertical for 5+ nodes, arrows ([67a6820](67a6820138a9649783aa716a907e46dc83481039))


## [0.4.35] - 2026-07-28

### <!-- 0 -->🚀 Features

- replace CSS flow diagrams with React Flow + ELK auto-layout ([0d9daca](0d9daca6c3cfc90744473e016934f539729e75a5))


## [0.4.34] - 2026-07-28

### <!-- 4 -->⚡ Performance

- cache auth inventory on backend, populate at startup ([2d5a4eb](2d5a4ebcda0496d9ff2a5e550d5740187008e407))


## [0.4.33] - 2026-07-28

### <!-- 4 -->⚡ Performance

- cache auth inventory in Zustand store, fetch once at startup ([4fc3ee9](4fc3ee965e190b5d35f60d9087efb94cf37f7561))


## [0.4.32] - 2026-07-28

### <!-- 0 -->🚀 Features

- add /api/health and /api/version endpoints ([1e62368](1e6236883d5653de2b6f4d1ce2abca3b91302f6c))


### <!-- 7 -->⚙️ Miscellaneous Tasks

- remove temporary Playwright review scripts ([787685d](787685d6c19bebe2d10acdccb2efe003d4c3a6b3))


## [0.4.31] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- flow diagram arrows, CF Access bypass detection, and node rendering ([ab20bc4](ab20bc48a9751ac8c12402d62555842a4d819f6a))


## [0.4.30] - 2026-07-28

### <!-- 0 -->🚀 Features

- add step-by-step request flow tables to VisualizeModal ([b269217](b269217d515e17bd7de7f8137e1e619233ab1c62))


## [0.4.29] - 2026-07-28

### <!-- 0 -->🚀 Features

- render auth steps as nodes in the flow diagram ([cbe74e0](cbe74e0ea6ece2eb2ebda3b13927a889401ef00d))


## [0.4.28] - 2026-07-28

### <!-- 0 -->🚀 Features

- add auth pattern analysis to VisualizeModal ([9d70adb](9d70adbcdeb0f75ef9444dddbb8b9d95b9e4ed8a))


## [0.4.27] - 2026-07-28

### <!-- 0 -->🚀 Features

- enrich VisualizeModal with auth inventory data (CF Access + Authentik) ([46bf1b8](46bf1b8621e29baee1bf59b2e2ac04ce734642c4))


## [0.4.26] - 2026-07-28

### <!-- 0 -->🚀 Features

- add per-entry flow visualization modal ([fc54ed7](fc54ed7c743092ab57744e04102fc2bba53c8447))


## [0.4.25] - 2026-07-28

### <!-- 0 -->🚀 Features

- add copy-to-clipboard button for caddy upstream values ([f89c373](f89c3731bbf52fcd4d432dcc3f38a1301c89cd92))


## [0.4.24] - 2026-07-28

### <!-- 7 -->⚙️ Miscellaneous Tasks

- add Telegram notifications for releases and CI failures ([6b85741](6b85741575011bd6ff740a17266b6fadfb2a9688))


## [0.4.23] - 2026-07-28

### <!-- 7 -->⚙️ Miscellaneous Tasks

- expand ESLint with React, TypeScript, and JS best-practice rules ([c219ee7](c219ee77f722de1f996c6b2cd57ab68457b839a1))

- add web-lint pre-commit hook for ESLint ([723b6e4](723b6e4abd5c9fd4dd56c8df6832f00b034c676d))

- add ESLint with custom rule to catch unstable Zustand selectors ([1a90640](1a90640198b93526a2bcf28fe16507205478330f))


## [0.4.22] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- infinite render loop from Zustand selectors returning new arrays ([89a8281](89a8281ae7abed62bc7c35b9c5671f407e492937))


## [0.4.21] - 2026-07-28

### <!-- 2 -->🚜 Refactor

- introduce Zustand store and split CSS into per-component files ([b504ae5](b504ae5604220dbcaaf5d890b000a5081d4bfb1b))


## [0.4.20] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- define missing --bg-1 CSS variable causing transparent dropdowns ([c81f7b0](c81f7b045518859ea7da73a67fe73162defb23e5))


## [0.4.19] - 2026-07-28

### <!-- 0 -->🚀 Features

- add prune stale entries endpoint and UI ([c3e552c](c3e552c793350d8e6dc0a79b72b0513f2d81ced9))


## [0.4.18] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- skip wildcard/root domain entries in diagnostics ([b954576](b9545766de7d3ba334df8479c57961e39d66200d))


## [0.4.17] - 2026-07-28

### <!-- 0 -->🚀 Features

- add /api/diagnostics endpoint and Diagnostics tab ([aa33172](aa3317211728e3719c377b92ec87a925f324336f))


## [0.4.16] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- sanitize trailing commas from Caddy host matchers ([704b43e](704b43e6508677af0afe0f92ffdcf0d4d1fd5be0))


## [0.4.15] - 2026-07-27

### <!-- 0 -->🚀 Features

- add deploy confirmation modal before applying auth changes ([4821b08](4821b085011806f40af2c7e5e91709730afb0da0))


## [0.4.14] - 2026-07-27

### <!-- 0 -->🚀 Features

- add inline auth editing UI with pending changes and edit modal ([2acc094](2acc0946988b3f46300796e1821aab8969c4c24d))


## [0.4.13] - 2026-07-27

### <!-- 0 -->🚀 Features

- add info tooltips to all auth table column headers ([ff62d4a](ff62d4a465f6ea4837a4426b59fb157ab1213813))


## [0.4.12] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- stop stretching auth table to full container width ([c7e4dfb](c7e4dfb89c3be1aba006a3ccd36215e811d3bf6e))


## [0.4.11] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- normalize auth table row heights ([8b1c7ab](8b1c7ab111c000ad926d18f6cb7fce029d1d044c))


## [0.4.10] - 2026-07-27

### <!-- 0 -->🚀 Features

- add search bar, tighten horizontal spacing, clarify N/A ([7070486](7070486fac664b7fccaf525d487d314b43173f9d))


## [0.4.9] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- tighten Auth Flows spacing and clarify API Auth column ([9d01e81](9d01e8177bdcd2855b35853e8f411ee8e12e0cee))


## [0.4.8] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- use Unlock icon for LAN "None" auth (consistent with WAN) ([78dae10](78dae10c5bdea2f412985f406d0ad26137f516fa))


## [0.4.7] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- change LAN "None" icon from WifiOff to DoorOpen ([34838ec](34838eca55b05ca8bb510405833d5435df559d64))


## [0.4.6] - 2026-07-27

### <!-- 0 -->🚀 Features

- stream dashboard entries loading via SSE with per-service progress ([970a29e](970a29eeee7277974d047b78cc2c56805aa186ff))


## [0.4.5] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- don't prematurely classify WAN auth before CF Access enrichment ([972b8be](972b8be100aa64defae0e16c90c9fe08bad3edc4))


## [0.4.4] - 2026-07-27

### <!-- 0 -->🚀 Features

- unique icons per auth type in table rows and headers ([9c5795c](9c5795c127df15b08757120757f3c382eef4cf94))


## [0.4.3] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- cache config in sessionStorage so version survives page reload ([5006a38](5006a38b62a6b9e4da216902d8fda947486c8427))


## [0.4.2] - 2026-07-27

### <!-- 0 -->🚀 Features

- visual auth legend with icons, color-coded sections, flow diagram ([0612e6d](0612e6deec47a7aff607bd8e6fefe045ae7589f7))


## [0.4.1] - 2026-07-27

### <!-- 0 -->🚀 Features

- stream auth inventory via SSE + add auth type legend ([3927dd5](3927dd5579df11b2cc1d64328abc9a2e3876c34e))


### <!-- 1 -->🐛 Bug Fixes

- use content-hashed asset filenames for cache-busting ([5831e9c](5831e9ce931686f26495229a2b42aef825ae08d8))


## [0.4.0] - 2026-07-27

### <!-- 0 -->🚀 Features

- add Auth Flows tab with Authentik + CF Access discovery ([f03a8e8](f03a8e8e8b71fb51eaac4cd8cd801cde9b5a3106))


## [0.2.2] - 2026-07-25

### <!-- 0 -->🚀 Features

- show build timestamp when version is "dev" ([c0ea9db](c0ea9dbca87e2b08bb83a36126902a2b7a057d9a))


## [0.2.1] - 2026-07-25

### <!-- 2 -->🚜 Refactor

- split Dashboard.tsx, add plan TTL, web API improvements ([e21a2ad](e21a2ad8e28b7b0aa6e233751f5dec387441f6f6))


## [0.2.0] - 2026-07-25

### <!-- 0 -->🚀 Features

- detect and warn on Authentik forward_auth bypass via CF tunnel ([110fa2c](110fa2c7848fc380159c9a417f1e1f61bc992f9a))

- add OriginServerName support to CF tunnel edit flow ([01770de](01770de6b9f357288382d4f818b466f5bb4fb395))

- mark-intentional toggle to suppress warnings per session ([b27d410](b27d410355e037239a5f8e74571632968dfebe92))

- add long-timeout, forward-auth, compression-no-tls-verify templates ([aae0e71](aae0e7108ca10004fd5d4810b7f5bb40f42e6ea9))

- show Caddy upstream health ([bf5b1ab](bf5b1ab963eac5948e3d12efa0d4c19d4b52ae72))

- global activity spinner in sticky topbar ([2aa5035](2aa5035aeba46ed490dd5de12f911752ddad0877))

- show build version in web UI topbar ([7c4ff3f](7c4ff3fb2c0d54454ec71acdd6d8aca01e0d5b31))

- update default Caddyfile template to use import proxy_headers ([359bc9c](359bc9c93e164f80fa9b1b9d6f0957e29c1a9b44))

- full status rewrite + CF repair-dns + repair banner ([dce2324](dce23241656d4bd0f685d7343b6271e8313d56e3))

- add spinner to Cloudflare route buttons while saving ([33f5771](33f5771c66774c5c052954620b67a29b67dd939f))

- add drag-to-resize handle on server log panel ([9321f5a](9321f5a7bfc3ee3342fb35ddacf5ae3f482180d2))

- log load summary with actionable warnings after each data refresh ([2a812e9](2a812e9f8e6b648ed6553394c514bf452e738b3f))

- surface CF missing CNAME as issue in dashboard ([8070bb4](8070bb43e989fd8684c2f96421ba2d5649279d56))

- CF DNS CNAME sync, DNS probe with auto-retry, deploy logging ([dfa1297](dfa1297a883d3f426f4ac38ffac0c69684d93560))

- refine collision detection UI with tooltip badges ([1a006bd](1a006bd251218ce3e70403be44befd6ce6a46dd4))

- merge collision detection + global git banner ([cb858c7](cb858c7fba1fa299edbd5a8a00e41de98a7db9ea))

- add hostname collision decision helper ([084c969](084c9696df0176d3eee3459a14143a3078088499))

- add direct Cloudflare tunnel hosts ([f0c0e78](f0c0e788f99749195ab2f494fc6fbd3c41977109))

- improve sync web and tui status ([20fdec8](20fdec886cbede0d295de8f355157efc833fc1ef))

- web tweaks ([b091c83](b091c834db4cd6af590d84f157257cde4bec8e90))

- add cloudflare tunnel sync planning ([96349d2](96349d2a9f4e66a714a42f3c995ea7531ca392a8))

- refactor web console ([ecb0c89](ecb0c89e04695484d8a18e16f22fd1f857c7b68a))

- tighten web app layout ([0f95ca5](0f95ca532743ebf6676809320c852b2792f6526a))

- convert web ui to react spa ([b167680](b167680cdd7eb3806e3e7a60b4cc1dde77e167ec))

- add web config tests ([f5b7ad8](f5b7ad84185e8de9e475dc323e2d34f8cf5e4188))

- clarify web loading progress ([6abd3cb](6abd3cb9be1166440d80c065547cb4496966acee))

- refine web dashboard UI ([cc73421](cc734211ccc815a3d0266c7efae896aae913eb46))

- support web config editing ([5c30d5c](5c30d5cbd44dc3f675135f4a713555d6819ff5eb))

- show sanitized web config summary ([a46fb5b](a46fb5b87946228f056b444596fae4f711556bd9))

- back web sync controls with apply API ([83c9a05](83c9a05b8e29b6c6b99839324e7d2656dc391878))

- improve web dashboard usability ([3d91896](3d918966f76e91040b1106ab6f7bcad983f59c2c))

- build browser UI workflow ([370459c](370459ccd61d91664640ac690841a8f7fa63c980))

- add shared runtime and web UI foundation ([454a1d3](454a1d3d7305c9ff89d57037426caebe6fb57d5d))

- implement CloudflareClient write methods (Step 2) ([75fd7bf](75fd7bf8a64a44efd4f5b8c9d54996a26762275a))

- Cloudflare tunnel config with interactive wizard and TUI improvements ([dbf6479](dbf64798251e39feebcc148949aeb8b697e4def0))

- bunch-o-changes - tui is mostly working ([09be061](09be06163373e6031a21dab2f54f404220888902))


### <!-- 1 -->🐛 Bug Fixes

- query Unbound DNS directly instead of system resolver ([3c01e6f](3c01e6f420f7fcef9285d2eee17a9baf1d3e6232))

- replace conflicting A/AAAA/CNAME records instead of erroring ([92a2fff](92a2fff9b5e5f150a49845dba7169ba8925cfa3b))

- metric card tones now reflect zero-state correctly ([ddf009d](ddf009d482efeedad3544e79ce443753f5d88d39))

- detect conflicting DNS records before CNAME creation in EnsureDNSRecord ([2ca82b9](2ca82b95a50e827eb9d7fbfac6b55b87f5433575))

- surface DNS CNAME failure as warning instead of silent success ([4a8dce3](4a8dce353bcfc5b9272c1bb8eb848228cc647bfc))

- delete companion matchers (e.g. @name_external) on entry removal ([bc5242f](bc5242f4ff47271d60b4057ed3b2dccb6b24b34c))

- wire Params through preview/UI and fix forward-auth parser ([e1fa30b](e1fa30b78ab14f0d10a09f775721fc13dd9dd26d))

- show web sync controls ([ac14fe9](ac14fe9807dd0c23e4598a75a134747343d7346a))


### <!-- 10 -->💼 Other

- redesign Modify modal, fix CF :80 stripping, add query cmd + exclude-hostnames ([ff4a59b](ff4a59bbc08e7c1766f53ebd5e09fe4386fa2fce))


### <!-- 3 -->📚 Documentation

- update plan.md with completed items and current status ([7f938d9](7f938d99fb45bae6d99d50da8da1c6fccd8cb662))


### <!-- 5 -->🎨 Styling

- polish web app density ([79c62fc](79c62fcfd5ef2750f660f0adde4ad4d8e9f154d6))


## [0.1.0] - 2025-09-15

### <!-- 0 -->🚀 Features

- adguard sync workign ([510717c](510717c28bccf91bc010546139ba9aa1f563cb0d))


### <!-- 1 -->🐛 Bug Fixes

- version template error ([5c358de](5c358defbb06da04f4cc5236a1c115a6c1c07543))


### <!-- 3 -->📚 Documentation

- update readme ([302ad10](302ad10d9618c015fa0df1cd7f6e810b46c1a062))


## [0.0.1] - 2025-05-06

### <!-- 0 -->🚀 Features

- initial commit ([1497ed9](1497ed9e87c5a06a2d7ddb82423ad9c24c33a43e))


<!-- generated by git-cliff -->
