# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### <!-- 0 -->🚀 Features

- add durable resource ownership state ([0ca805d](0ca805d4db7016a67e5dd75285ce618d04ed08cb))


### <!-- 1 -->🐛 Bug Fixes

- preserve cloudflare catch-all rules ([f454ba0](f454ba0eab112bccfe3003be19b5f2711ccf2acd))

- invalidate web plans after config changes ([f4704ae](f4704aed022254ae95bdfe28159efd48d48742ed))

- require managed unbound records for mutations ([b80b766](b80b766481a793c81e595b5709ba046e9ee314c9))

- protect direct web record removal ([d3de1ec](d3de1ec9e9ddb87bbff50f523481b6815be9a770))

- require fresh inventory for sync plans ([6e51a96](6e51a964c9728c58d77002540fae4dea6956ef53))

- honor selected config in cloudflare sync ([1bc4c62](1bc4c62962df58e3dc6d19fb2fa73fef2f3c2923))

- claim web sync plans before apply ([0b15378](0b153786463b9bd587bcb4512ed9e6ae86eebe60))

- enforce explicit adguard ownership ([71a6470](71a64705c1ccf4b145bb6b09fd74e329f85d4dd5))

- make caddy upstream edits transactional ([15fdbf7](15fdbf7c562bc6efd4dd8973fc7b1fe438c00781))

- honor selected caddy endpoint everywhere ([2f1fe2c](2f1fe2cbfcc5586f13e97505be0b530e8c4c517e))

- resolve effective configuration safely ([4c3a664](4c3a664f952079151bc9dacb21e040f79ca9d496))

- harden web access and probe policy ([46bb98a](46bb98a17e948f8d5fb65c28db1570930ece1a88))

- satisfy frontend end-of-file check ([086dea9](086dea9783511c653dc0c5f7f1f781576202036d))

- sync button 502s and file_server hostnames skipped ([18f4b6b](18f4b6bc812c30d953011b94cf2e20a28946acee))

- separate Caddy admin API host from LAN IP for DNS comparison ([4e006d6](4e006d6f4891b8c0993ab7781e2cd1e416ceee89))

- correct service name and two-step copy in Makefile deploy targets ([344a805](344a805c85e92fe0a6c314fdaa3d724501cf7201))


### <!-- 3 -->📚 Documentation

- update changelog ([6d71695](6d71695a522a66f38f2c5baf79efeb96601e88aa))

- update changelog ([3f8ae94](3f8ae943b711b503305e300a11caf7d02cba25be))

- update changelog ([6314f4d](6314f4d622c19a885ce3fcfd6011c84e47f67284))

- update changelog ([0c38087](0c38087ce5e28f55d71e6a45de97736697cc05f1))

- update changelog ([fbfbaa1](fbfbaa1650fcffe9b7a6f729723d2101b0a49c74))

- update changelog ([6828d5d](6828d5dc765080b156ea875a31f7d5f858a73efc))

- update changelog ([7c6c9cc](7c6c9cc69da80881d1929fe71d044396d146debe))

- update changelog ([b389120](b3891204c89486ced6a85ed9749bddee821f9d6c))

- update changelog ([f6b1b77](f6b1b77cacad13dd57ffb5b2c0620d3443f185f2))

- update changelog ([d8cda1d](d8cda1de2704d2d6eddf3e5b7aebef3a7edc926c))

- update changelog ([ea443fb](ea443fb8758a25e086333593f6ec383b3c859380))

- update changelog ([f01519f](f01519f54d61c0d6449ce333e5c8d3b064ffd44b))

- update changelog ([c58650d](c58650dcf32ff29bb537de4674f5197bb73f116d))

- update changelog ([a85a191](a85a191887a6c5717c4e3079e50ca8efa71ab067))

- update changelog ([48d3233](48d32330879f867f044f5bf68376963dc8c7ddeb))

- update changelog ([244dce8](244dce83fb4d0dd8b94c9c590f48d499ad5da1ec))

- update changelog ([ff33775](ff33775e79ef98aeb46c43bb7b17a97a267ef6ee))

- update changelog ([e8bc99c](e8bc99c555b070e4aecf690b3fc3476af78f8a58))

- update changelog ([638337c](638337ccce4ac4d9bc0f2d5df12d0a3cf9739c6b))

- update changelog ([e6705d4](e6705d4030fd633aca1c346e5c1a601bfe2d93c7))

- update changelog ([d3fdab5](d3fdab55dbcf087ab4d603403388a3e1cca6a238))

- update changelog ([e3096ae](e3096ae3722951f1aab3ee97b6242e7d1416d3ec))


### <!-- 6 -->🧪 Testing

- strengthen web verification foundation ([1b985ff](1b985ff334466e2b5c724a140894037cf4630334))


### <!-- 7 -->⚙️ Miscellaneous Tasks

- update govulncheck action runtime ([d5bc79d](d5bc79d522550eee0e9d3a4e0282376a26c7bf8f))

- update CodeQL upload action runtime ([b969c81](b969c81c3c9ff8d4d2753fc563e1f31b648f5b68))

- move GitHub Actions to Node 24 ([08ac2a5](08ac2a500e9535b6c5b6ade317c3f4e1cd54d726))

- add missing .PHONY targets and fix help text alignment ([3e0a022](3e0a022e549ef570a61c3a72e0b8b5a2af07fbe1))


## [0.5.2] - 2026-09-02

### <!-- 0 -->🚀 Features

- selective prune with checkboxes + Prune All button ([dffaa85](dffaa85b59fbaa153de336093e7ef83f9af4aae2))

- selective prune with checkboxes + Prune All button ([1ee9c94](1ee9c94ee9a9e5e6724c0437495b8133d29312bd))

- add fix-double-login action to create CF Access bypass ([4c4bdf7](4c4bdf7e3af3a31a53c12aa9e07e8342eb63d3e8))

- cosign-sign Go binary, add CODEOWNERS, Scorecard, Go 1.26 ([886e0dc](886e0dc1cbabe4a6215a0992ab0e164881526003))

- add SBOM workflow with SPDX + CycloneDX ([4590a68](4590a68ca84cbcdc1144a88fcc5ca5f0617f30bf))

- add sb.vookie.net to forward_auth registry ([649c6cc](649c6cca39b65ccf12737fba986d0594aeb9cc09))

- add forward_auth registry to detect missing auth regressions ([66c9df2](66c9df25888e3295a1f98d7c3e97fda20e51064f))

- graceful shutdown, SSE diagnostics stream, repair-dns progress ([0c98416](0c98416c7df80b747677149fd4ddd14b6f3047f3))

- auto-detect Authentik IdP host to avoid false positive ([6e51aad](6e51aad71bbb3a44e9fcae82cd82d48a82ef4ed8))

- standalone visualize page at /visualize/{hostname} ([5ffe119](5ffe1193c0bfa784f681f10b170b4b97aca6b9f6))

- deep link support for visualize modal ([11298d9](11298d92b7337956908c563503a9592e638a0338))

- show service name instead of 'Service' in diagram nodes ([7745de5](7745de562d4168367bf0db75af419d3f44dbae49))

- side-by-side diagram+flow layout, custom arrow edges, step numbers ([2d55515](2d555158278b80d879b104ea22c420270d0b25fb))

- replace CSS flow diagrams with React Flow + ELK auto-layout ([0d9daca](0d9daca6c3cfc90744473e016934f539729e75a5))

- add /api/health and /api/version endpoints ([1e62368](1e6236883d5653de2b6f4d1ce2abca3b91302f6c))

- add step-by-step request flow tables to VisualizeModal ([b269217](b269217d515e17bd7de7f8137e1e619233ab1c62))

- render auth steps as nodes in the flow diagram ([cbe74e0](cbe74e0ea6ece2eb2ebda3b13927a889401ef00d))

- add auth pattern analysis to VisualizeModal ([9d70adb](9d70adbcdeb0f75ef9444dddbb8b9d95b9e4ed8a))

- enrich VisualizeModal with auth inventory data (CF Access + Authentik) ([46bf1b8](46bf1b8621e29baee1bf59b2e2ac04ce734642c4))

- add per-entry flow visualization modal ([fc54ed7](fc54ed7c743092ab57744e04102fc2bba53c8447))

- add copy-to-clipboard button for caddy upstream values ([f89c373](f89c3731bbf52fcd4d432dcc3f38a1301c89cd92))

- add prune stale entries endpoint and UI ([c3e552c](c3e552c793350d8e6dc0a79b72b0513f2d81ced9))

- add /api/diagnostics endpoint and Diagnostics tab ([aa33172](aa3317211728e3719c377b92ec87a925f324336f))

- add deploy confirmation modal before applying auth changes ([4821b08](4821b085011806f40af2c7e5e91709730afb0da0))

- add inline auth editing UI with pending changes and edit modal ([2acc094](2acc0946988b3f46300796e1821aab8969c4c24d))

- add info tooltips to all auth table column headers ([ff62d4a](ff62d4a465f6ea4837a4426b59fb157ab1213813))

- add search bar, tighten horizontal spacing, clarify N/A ([7070486](7070486fac664b7fccaf525d487d314b43173f9d))

- stream dashboard entries loading via SSE with per-service progress ([970a29e](970a29eeee7277974d047b78cc2c56805aa186ff))

- unique icons per auth type in table rows and headers ([9c5795c](9c5795c127df15b08757120757f3c382eef4cf94))

- visual auth legend with icons, color-coded sections, flow diagram ([0612e6d](0612e6deec47a7aff607bd8e6fefe045ae7589f7))

- stream auth inventory via SSE + add auth type legend ([3927dd5](3927dd5579df11b2cc1d64328abc9a2e3876c34e))

- add Auth Flows tab with Authentik + CF Access discovery ([f03a8e8](f03a8e8e8b71fb51eaac4cd8cd801cde9b5a3106))

- show build timestamp when version is "dev" ([c0ea9db](c0ea9dbca87e2b08bb83a36126902a2b7a057d9a))

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

- adguard sync workign ([510717c](510717c28bccf91bc010546139ba9aa1f563cb0d))

- initial commit ([1497ed9](1497ed9e87c5a06a2d7ddb82423ad9c24c33a43e))


### <!-- 1 -->🐛 Bug Fixes

- write CaddyServerIP to DNS instead of upstream target ([a798739](a7987395c6682531e0f8d9dd1e1833351b47759c))

- remove dead Sidebar.css, memoize StatusChip, add ErrorBoundary to lazy route ([30e62ac](30e62ac3cae10b94b97fc300e1f4915e3c9840a5))

- modal accessibility, remove dead Sidebar, log Go write errors ([ef623f0](ef623f023241df67116a413e7668ae526cbee8e1))

- Go backend bugs, dead code removal, bundle splitting, CI/CD updates ([28eb015](28eb0155e7a7b768689f4a91fcd3351d2a575fca))

- resolve all 138 eslint warnings ([5c90c75](5c90c7587799c9bda5b808f6961f84bdd8364e72))

- remove dead Sidebar.css, memoize StatusChip, add ErrorBoundary to lazy route ([9465228](94652287a6f5dde5216141d6f3b63209b5ca7a6c))

- modal accessibility, remove dead Sidebar, log Go write errors ([d32f97b](d32f97b0c43f7d293fbb712dc0bdc8d068396bad))

- Go backend bugs, dead code removal, bundle splitting, CI/CD updates ([52d989d](52d989d2c435253a7b57e628895982f2d2295806))

- resolve all 138 eslint warnings ([fc27a84](fc27a84eed0cc9d5988efd6884f484bf9d7b396b))

- add --repo flag to gh release upload in cosign-sign ([3d2d20b](3d2d20b121ce7cf67a292a20c2c53aec7eadab8a))

- use ./*.glob in release upload to satisfy shellcheck ([0316774](0316774ad062ea587d61b4c89fcccb50f850d38f))

- fix evaluated-envs format for SLSA builder ([6f8f9af](6f8f9affeba0b08f6c32c281b7b2c27dad172bef))

- add evaluated-envs to SLSA Go builder ([a118af2](a118af2b218b620a5649ea460d7ebbf951574ada))

- skip internal/status tests (require OPNSense/Caddy server) ([2cbf174](2cbf17480d01057ad820cc676336e0757035b16d))

- install Caddy for caddyeditor tests, skip race test ([d310be7](d310be71b95c66980839658200f89f2cc92d51f0))

- fix EOF newlines in store.ts and CHANGELOG.md ([e680c51](e680c517f303a0cb5ab36b0ad1b3603adc1f11f2))

- fix trailing whitespace in built/static files, changelog push perms ([553f9f6](553f9f61976e8795af2b155d5c72a58791b03fd9))

- fix pre-commit formatting issues and changelog workflow ([65582b2](65582b29e8453cac7d445fd01c24570050cf04c8))

- fix cliff.toml for git-cliff v4 compatibility ([15b4839](15b4839352c866ca17e3243b6f5728d76928f6f5))

- fix shellcheck word splitting, upgrade git-cliff-action ([28d27d4](28d27d4eac820689369ce4d0d5a513929fb300f5))

- make trivy non-blocking, fix npm ci peer dep conflict in build/lint ([1435dba](1435dba553f475971f3bc9d2261ec924775aa029))

- replace broken trivy-action, fix govulncheck and npm ci ([710cd11](710cd11683d9964983079a50659d172f58dc226f))

- concurrency control, security hardening, and reliability improvements ([c184cb9](c184cb9eda8652f2fdd2c8ea76f270348ce1007d))

- clean up error handling, validation, and minor code quality issues ([9b67f16](9b67f16d2941c87dc0d32c657773a329fbff8d46))

- critical build, CI, and code quality issues ([1bad4de](1bad4deeb866c774dd87d0d4a528a74c1288def9))

- flag WAN-exposed hosts with CF Access bypass-only as critical error ([e571e6c](e571e6cac415f8d7198038dac5989e31f2c49b58))

- detect conditional forward_auth to avoid false double-login warning ([eba90a6](eba90a6c507683954be2ef0053160b5f87007da9))

- increase ELK node spacing to reduce cramped layout ([173e236](173e236d98e8ed17b5aeb2531ab3c7ee88da1a85))

- always use vertical layout for consistency ([6ea64cd](6ea64cdef4bc4f71f3eef15756fac3ccbe069db0))

- arrows now visible, move IdP label to JWT verified ([4a26a59](4a26a59aff16b47a883c55ae1986fea89529005a))

- React Flow diagrams — fitView after layout, vertical for 5+ nodes, arrows ([67a6820](67a6820138a9649783aa716a907e46dc83481039))

- flow diagram arrows, CF Access bypass detection, and node rendering ([ab20bc4](ab20bc48a9751ac8c12402d62555842a4d819f6a))

- infinite render loop from Zustand selectors returning new arrays ([89a8281](89a8281ae7abed62bc7c35b9c5671f407e492937))

- define missing --bg-1 CSS variable causing transparent dropdowns ([c81f7b0](c81f7b045518859ea7da73a67fe73162defb23e5))

- skip wildcard/root domain entries in diagnostics ([b954576](b9545766de7d3ba334df8479c57961e39d66200d))

- sanitize trailing commas from Caddy host matchers ([704b43e](704b43e6508677af0afe0f92ffdcf0d4d1fd5be0))

- stop stretching auth table to full container width ([c7e4dfb](c7e4dfb89c3be1aba006a3ccd36215e811d3bf6e))

- normalize auth table row heights ([8b1c7ab](8b1c7ab111c000ad926d18f6cb7fce029d1d044c))

- tighten Auth Flows spacing and clarify API Auth column ([9d01e81](9d01e8177bdcd2855b35853e8f411ee8e12e0cee))

- use Unlock icon for LAN "None" auth (consistent with WAN) ([78dae10](78dae10c5bdea2f412985f406d0ad26137f516fa))

- change LAN "None" icon from WifiOff to DoorOpen ([34838ec](34838eca55b05ca8bb510405833d5435df559d64))

- don't prematurely classify WAN auth before CF Access enrichment ([972b8be](972b8be100aa64defae0e16c90c9fe08bad3edc4))

- cache config in sessionStorage so version survives page reload ([5006a38](5006a38b62a6b9e4da216902d8fda947486c8427))

- use content-hashed asset filenames for cache-busting ([5831e9c](5831e9ce931686f26495229a2b42aef825ae08d8))

- query Unbound DNS directly instead of system resolver ([3c01e6f](3c01e6f420f7fcef9285d2eee17a9baf1d3e6232))

- replace conflicting A/AAAA/CNAME records instead of erroring ([92a2fff](92a2fff9b5e5f150a49845dba7169ba8925cfa3b))

- metric card tones now reflect zero-state correctly ([ddf009d](ddf009d482efeedad3544e79ce443753f5d88d39))

- detect conflicting DNS records before CNAME creation in EnsureDNSRecord ([2ca82b9](2ca82b95a50e827eb9d7fbfac6b55b87f5433575))

- surface DNS CNAME failure as warning instead of silent success ([4a8dce3](4a8dce353bcfc5b9272c1bb8eb848228cc647bfc))

- delete companion matchers (e.g. @name_external) on entry removal ([bc5242f](bc5242f4ff47271d60b4057ed3b2dccb6b24b34c))

- wire Params through preview/UI and fix forward-auth parser ([e1fa30b](e1fa30b78ab14f0d10a09f775721fc13dd9dd26d))

- show web sync controls ([ac14fe9](ac14fe9807dd0c23e4598a75a134747343d7346a))

- version template error ([5c358de](5c358defbb06da04f4cc5236a1c115a6c1c07543))


### <!-- 10 -->💼 Other

- replace bottom tables with pill badges ([2002c71](2002c71fe2d15ace1ecb9dc33b2cfb52fae5278d))

- redesign Modify modal, fix CF :80 stripping, add query cmd + exclude-hostnames ([ff4a59b](ff4a59bbc08e7c1766f53ebd5e09fe4386fa2fce))


### <!-- 2 -->🚜 Refactor

- extract reusable components, fix bugs, consolidate CSS ([aad4029](aad4029b7b7a7c6d7c511f1bff17b812af117525))

- extract reusable components, fix bugs, consolidate CSS ([425034c](425034c7016045a732854f508f5e95bfab025932))

- improve error handling, validation, and code organization ([5a2cad7](5a2cad7f9f9704826e18b99d0ab57d10949dcf1e))

- split monolithic server.go into domain-specific files ([78d0563](78d0563cf5b53262b2933585b4dd6e940648a167))

- deprecate conditional forward_auth (Pattern E) ([3e59df0](3e59df0cb32b6ec16b8d7ee7ca646d380ba6ec41))

- replace flow table with vertical step list ([d83cbb0](d83cbb05df4a8164ab6da5e7f166604c959a7a4c))

- group WAN diagram+flow together, LAN diagram+flow together ([9a33005](9a33005d5b0c6d14592b03e092dd30d9bb898d31))

- replace status tile grids with config tables ([dfaef19](dfaef192bb3d17c524c679857430ecbc29f42933))

- introduce Zustand store and split CSS into per-component files ([b504ae5](b504ae5604220dbcaaf5d890b000a5081d4bfb1b))

- split Dashboard.tsx, add plan TTL, web API improvements ([e21a2ad](e21a2ad8e28b7b0aa6e233751f5dec387441f6f6))


### <!-- 3 -->📚 Documentation

- update changelog ([001b5db](001b5db321b3a51a786c81153b6d18a4ae6d9943))

- update changelog ([03f4c35](03f4c356c60b65022d600e231e89ca375cdca1fa))

- update changelog ([f8aa6c2](f8aa6c25db6f14dceccd8e1223fa969828f53de6))

- update changelog ([f298776](f29877602668377c5900077c3a6e428320ada0c0))

- update changelog ([b15dc1b](b15dc1b91753ff4b480bad23be3047c5d2b6f232))

- update changelog ([b2f169d](b2f169de8ffe96c2279189f29faca5e4bac1f609))

- update changelog ([571f7ad](571f7ad5ba71812db8f6dd363f0846d2fceffd6d))

- update changelog ([fe1dfe2](fe1dfe26f25da89b8570d40f9afae37ec43e064e))

- update changelog ([80ea0c4](80ea0c4202be50a7e897ef3eb23f95336ba46997))

- update changelog ([cb5d554](cb5d554a9d0a41dced290317e02b980855e2c805))

- update changelog ([b19ada6](b19ada6981e931dfb7952f3b868cd0ea987299e6))

- update changelog ([6faf806](6faf806961fca4708f1d13023abdfd44dc1d2310))

- update changelog ([84ed5c9](84ed5c9d03c28fac242a3aa80d9c52b4838a64f4))

- update changelog ([459e3c9](459e3c91cf28d6311d83ba05923f90cbbd8873ef))

- update changelog ([05117f8](05117f8a4a147cec1d841b270f3b01f4a51cd45d))

- update changelog ([f5a80bb](f5a80bb4e7bde04653531de6e8095da80911808c))

- update changelog ([add8cfe](add8cfe4f9e707505fcdae61b559d2071b9f0d76))

- update changelog ([ba5a18a](ba5a18a86f46ebd30bfc00057369116a4e347249))

- update changelog ([0149918](01499184be190678b56863205e875c72ded47431))

- update changelog ([e84d600](e84d60052edf29fab42fefbfe0c61ec9f1b16827))

- update changelog ([25fca6b](25fca6b68b30baa8ef51b63096dcbf99f0193058))

- update changelog ([2d1e77a](2d1e77a005977bd50dc4425ba690b46a55840470))

- update changelog ([84a7a02](84a7a020382fc782482289ea9b16dde19d8a2c48))

- update changelog ([b84171a](b84171a1447e86ef425a983cbf6ac1cc090b779b))

- update changelog ([da99af8](da99af8b566ff6ba21e70795f06415329d18b516))

- update changelog ([8dbf817](8dbf8178c05d3692d1dc7ffcf001024aa2886b4b))

- update changelog ([8d0ea68](8d0ea688d3f8139a4dcde4ff9a39178395c85528))

- update changelog ([a9b2345](a9b234537538775f436efa0e131f29fe77aaa9d0))

- update plan.md with completed items and current status ([7f938d9](7f938d99fb45bae6d99d50da8da1c6fccd8cb662))

- update readme ([302ad10](302ad10d9618c015fa0df1cd7f6e810b46c1a062))


### <!-- 4 -->⚡ Performance

- memoize remaining list-rendered components ([bfa0a80](bfa0a8085724404ed5d2f3b203d275ca376db018))

- memoize remaining list-rendered components ([c20f033](c20f0335a22591d86f34f032c69e00f82ec7a38d))

- add context propagation, panic recovery, and entries caching ([2ab1889](2ab1889d789982f383b016da58734217fbee4976))

- cache auth inventory on backend, populate at startup ([2d5a4eb](2d5a4ebcda0496d9ff2a5e550d5740187008e407))

- cache auth inventory in Zustand store, fetch once at startup ([4fc3ee9](4fc3ee965e190b5d35f60d9087efb94cf37f7561))


### <!-- 5 -->🎨 Styling

- white circle badges with blue outline and blue number ([d536ad0](d536ad0a73f016744602ec05e83421f67354a09e))

- widen VisualizeModal from 720px to 1100px ([50e2a4b](50e2a4b7cf8da06ef7ce7391f1a0300cf5857ce2))

- polish web app density ([79c62fc](79c62fcfd5ef2750f660f0adde4ad4d8e9f154d6))


### <!-- 7 -->⚙️ Miscellaneous Tasks

- migrate to @eslint-react/eslint-plugin for ESLint 10 support ([a1daf5c](a1daf5c752de9bfde3416bc7b9d4d2e12ca3f684))

- migrate to @eslint-react/eslint-plugin for ESLint 10 support ([4f619d9](4f619d9641b0d9abac8a068466e823b5b29fa239))

- fix branch refs, add badges, CodeRabbit, fix Build/Lint ([f63ef03](f63ef03a8cf7d5d03dcb19015bd5fc6361b4a963))

- replace Dependabot with Renovate ([fe5fad7](fe5fad747ec4efca166506164a08d13a0d238541))

- add CVE scanning, Dependabot, SECURITY.md, enable gosec ([2bb3287](2bb3287d7e8aa81249b0aa9a23506e6701855dd1))

- remove temporary Playwright review scripts ([787685d](787685d6c19bebe2d10acdccb2efe003d4c3a6b3))

- add Telegram notifications for releases and CI failures ([6b85741](6b85741575011bd6ff740a17266b6fadfb2a9688))

- expand ESLint with React, TypeScript, and JS best-practice rules ([c219ee7](c219ee77f722de1f996c6b2cd57ab68457b839a1))

- add web-lint pre-commit hook for ESLint ([723b6e4](723b6e4abd5c9fd4dd56c8df6832f00b034c676d))

- add ESLint with custom rule to catch unstable Zustand selectors ([1a90640](1a90640198b93526a2bcf28fe16507205478330f))


## [0.5.1] - 2026-08-09

### <!-- 0 -->🚀 Features

- add fix-double-login action to create CF Access bypass ([84b202f](84b202fa39cb056010679406c242c1b60e1372d9))

- cosign-sign Go binary, add CODEOWNERS, Scorecard, Go 1.26 ([4e1d8bd](4e1d8bde169b3393f60da3d9e11809e77912bbd8))


### <!-- 1 -->🐛 Bug Fixes

- add --repo flag to gh release upload in cosign-sign ([3bdb755](3bdb7555a68be520deca12a784ea7e5b7442a85b))

- use ./*.glob in release upload to satisfy shellcheck ([5363fda](5363fdac59c2e97aacac557b2ccde5feb4f697dc))


### <!-- 3 -->📚 Documentation

- update changelog ([a02edec](a02edec92fc0cf4eb9787e23f357da8bde0abc7b))

- update changelog ([ea5f7b4](ea5f7b4f19a164e06ab5cf6f892a9b034a8925d0))

- update changelog ([f1af52e](f1af52e35e41229a4307862b4fba8f8ae7f7dceb))

- update changelog ([0fd2f42](0fd2f424faea815d7a41a612a3656f07cb5f0831))


## [0.5.0] - 2026-08-06

### <!-- 0 -->🚀 Features

- add SBOM workflow with SPDX + CycloneDX ([a2f2096](a2f2096a4b79943ccdba002fd0d7458911e523d4))

- add sb.vookie.net to forward_auth registry ([6352902](6352902b74ab3c9002c997eb25ad1fa079a19fc5))

- add forward_auth registry to detect missing auth regressions ([61e1c53](61e1c53ba1eb47e3e95349b7efb5f2fcf7384be9))


### <!-- 1 -->🐛 Bug Fixes

- fix evaluated-envs format for SLSA builder ([2dc5036](2dc5036db2173bc4766dbf80b8850c2d7ab9169c))

- add evaluated-envs to SLSA Go builder ([922ff8f](922ff8fb633628794027eeb303111ffb3f2dd5ca))

- skip internal/status tests (require OPNSense/Caddy server) ([7fef7eb](7fef7eb956d6dd5ff64afe60ced611d7d29a598c))

- install Caddy for caddyeditor tests, skip race test ([c94a2ca](c94a2cafa5adb032ddf305d7d6b2b91c099f75a8))

- fix EOF newlines in store.ts and CHANGELOG.md ([e300149](e3001499191af241da2b1aa8ff86a69401597326))

- fix trailing whitespace in built/static files, changelog push perms ([fa46fc1](fa46fc1ad2b1ea38d08409aaf9db49786c505e65))

- fix pre-commit formatting issues and changelog workflow ([f1e36c3](f1e36c3b00d09b383c7d78e5a795c347f196640a))

- fix cliff.toml for git-cliff v4 compatibility ([0b66d0d](0b66d0ded4ff94d463a55748c5b7ede7c0f8a618))

- fix shellcheck word splitting, upgrade git-cliff-action ([6d6a118](6d6a118ccfc2fc599d8dd599cc31f8922506ade8))

- make trivy non-blocking, fix npm ci peer dep conflict in build/lint ([dc63bb3](dc63bb3bbb3016027c93faa1da77626c2bbbce0c))

- replace broken trivy-action, fix govulncheck and npm ci ([d02fd7c](d02fd7c48bf737e9cd38ffe40c253f4ef6479693))

- concurrency control, security hardening, and reliability improvements ([67ba593](67ba59310f4c4abe0c61d968cfa7770e239e1448))


### <!-- 3 -->📚 Documentation

- update changelog ([d5bb2f6](d5bb2f6b5c434014eba00d2d24a8661b082d8489))

- update changelog ([5b959a2](5b959a248d0eb109d00be568ef0cd046006ce8ae))

- update changelog ([f5849e3](f5849e3824c169497d940624df36a67f70e28894))

- update changelog ([6e0d014](6e0d0142ef5c7e6b13b25d9b8678a638d5c3f0ae))

- update changelog ([f8151da](f8151da250f03805dc8f1e756b0e2859f4354cdb))

- update changelog ([849b9aa](849b9aad1a93273de224ec91b65d355844ff9a57))


### <!-- 7 -->⚙️ Miscellaneous Tasks

- fix branch refs, add badges, CodeRabbit, fix Build/Lint ([ab9eddf](ab9eddf0939d9ca64621058ba62fddb3674462e2))

- replace Dependabot with Renovate ([2d3adb7](2d3adb70abb30e6d70e946d6d110b2b9f5c971fe))

- add CVE scanning, Dependabot, SECURITY.md, enable gosec ([8af3521](8af3521e361d23e1782557785da2a27ae4520e41))


## [0.4.54] - 2026-07-29

### <!-- 0 -->🚀 Features

- graceful shutdown, SSE diagnostics stream, repair-dns progress ([d37455e](d37455eb967ff6d89d1934fd7ed648cf2815f730))


### <!-- 4 -->⚡ Performance

- add context propagation, panic recovery, and entries caching ([a997ab2](a997ab2ec6908982a7d1d2a8442efe50e5e08f2a))


## [0.4.53] - 2026-07-29

### <!-- 1 -->🐛 Bug Fixes

- clean up error handling, validation, and minor code quality issues ([e042aa0](e042aa085eefa012028b56af98f7290e3b3e5995))


## [0.4.52] - 2026-07-29

### <!-- 2 -->🚜 Refactor

- improve error handling, validation, and code organization ([7aadf3f](7aadf3ff85fbf2385701887a27996ebdc118b1ca))


## [0.4.51] - 2026-07-29

### <!-- 0 -->🚀 Features

- auto-detect Authentik IdP host to avoid false positive ([3a25ce6](3a25ce6cc4f49009274cc72f99a50c67aa144663))


### <!-- 1 -->🐛 Bug Fixes

- critical build, CI, and code quality issues ([746fce3](746fce361a75925a1fb7830eebcbc623c7411b3e))

- flag WAN-exposed hosts with CF Access bypass-only as critical error ([fb30fdf](fb30fdfb21b6159d2a71486d16cc675b5914e388))


### <!-- 2 -->🚜 Refactor

- split monolithic server.go into domain-specific files ([5270cd6](5270cd64d8dcada82eaf413a45f8c36547e1f41a))

- deprecate conditional forward_auth (Pattern E) ([8062f17](8062f179c3a925e8eac97b5d77d9e8055293061a))


## [0.4.50] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- detect conditional forward_auth to avoid false double-login warning ([de19e85](de19e8539fd6555a88bbfea87c0e84fa1c2bc167))


## [0.4.49] - 2026-07-28

### <!-- 0 -->🚀 Features

- standalone visualize page at /visualize/{hostname} ([89b3e9c](89b3e9cd9de7e58ad4ae24dcd8bbc5a57d4652f1))


## [0.4.48] - 2026-07-28

### <!-- 0 -->🚀 Features

- deep link support for visualize modal ([7e344c1](7e344c13c3bbaad96b03123df0ba0986dd4c0d51))


## [0.4.47] - 2026-07-28

### <!-- 10 -->💼 Other

- replace bottom tables with pill badges ([d61a53c](d61a53cf73deccf1c553320b1593a437412ed629))


## [0.4.46] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- increase ELK node spacing to reduce cramped layout ([a75aa1c](a75aa1c2880544c276e8611ec5da824b44a78799))


## [0.4.45] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- always use vertical layout for consistency ([3cc14ca](3cc14ca92da23b835f648aedd8aafe7d04deb090))


## [0.4.44] - 2026-07-28

### <!-- 0 -->🚀 Features

- show service name instead of 'Service' in diagram nodes ([f8c0ff4](f8c0ff474171d223c70b405347093d09aea86ca1))


## [0.4.43] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- arrows now visible, move IdP label to JWT verified ([1c060b8](1c060b8d09c409734514aecb54739e39e15a10f3))


## [0.4.42] - 2026-07-28

### <!-- 5 -->🎨 Styling

- white circle badges with blue outline and blue number ([bc3ec73](bc3ec730e4550ddb51be84f395e282a3c677d2c8))


## [0.4.41] - 2026-07-28

### <!-- 2 -->🚜 Refactor

- replace flow table with vertical step list ([3ca709f](3ca709f9acc6615c4b819f43f7b4b71bafbb27bb))


## [0.4.40] - 2026-07-28

### <!-- 0 -->🚀 Features

- side-by-side diagram+flow layout, custom arrow edges, step numbers ([bb04a5e](bb04a5eba1ca4288b0424271bf4f707979a56a21))


## [0.4.39] - 2026-07-28

### <!-- 5 -->🎨 Styling

- widen VisualizeModal from 720px to 1100px ([b8b1f13](b8b1f138b2da48921297f10ffe69117123a23c96))


## [0.4.38] - 2026-07-28

### <!-- 2 -->🚜 Refactor

- group WAN diagram+flow together, LAN diagram+flow together ([4f64b09](4f64b098af648674553d6d1c90a1a70c9bc351b4))


## [0.4.37] - 2026-07-28

### <!-- 2 -->🚜 Refactor

- replace status tile grids with config tables ([dea3d1c](dea3d1ce6c198b864b1eb66561b43357449906dc))


## [0.4.36] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- React Flow diagrams — fitView after layout, vertical for 5+ nodes, arrows ([fa0d892](fa0d892c709b1652d19a272ebb2444d5afe75bde))


## [0.4.35] - 2026-07-28

### <!-- 0 -->🚀 Features

- replace CSS flow diagrams with React Flow + ELK auto-layout ([a4814b2](a4814b2306b1f0f71431ed4564e9dede48ccd340))


## [0.4.34] - 2026-07-28

### <!-- 4 -->⚡ Performance

- cache auth inventory on backend, populate at startup ([c93a69c](c93a69cef0433dc39290d94aa370c63db9821276))


## [0.4.33] - 2026-07-28

### <!-- 4 -->⚡ Performance

- cache auth inventory in Zustand store, fetch once at startup ([b283bef](b283befda72f74c5f4ac8ff7b4076571b8d21f11))


## [0.4.32] - 2026-07-28

### <!-- 0 -->🚀 Features

- add /api/health and /api/version endpoints ([15354c3](15354c3e91f1d7b783c76ec0e8a552426a8c152b))


### <!-- 7 -->⚙️ Miscellaneous Tasks

- remove temporary Playwright review scripts ([83ea16d](83ea16d856d558217f5b15ef1ec7ab72ac2f3085))


## [0.4.31] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- flow diagram arrows, CF Access bypass detection, and node rendering ([890905c](890905c33a2df93944369ebbfac83c1e53ba074a))


## [0.4.30] - 2026-07-28

### <!-- 0 -->🚀 Features

- add step-by-step request flow tables to VisualizeModal ([e1c281b](e1c281be73f2c195a2a42570ee8248ec1e1e598f))


## [0.4.29] - 2026-07-28

### <!-- 0 -->🚀 Features

- render auth steps as nodes in the flow diagram ([77a1eb4](77a1eb4cb7e049a60d69d2d47f795bc4101e6478))


## [0.4.28] - 2026-07-28

### <!-- 0 -->🚀 Features

- add auth pattern analysis to VisualizeModal ([39ae936](39ae9365d1dbca1efc489a519f0d6df586928870))


## [0.4.27] - 2026-07-28

### <!-- 0 -->🚀 Features

- enrich VisualizeModal with auth inventory data (CF Access + Authentik) ([635d87f](635d87fe915b6455c8fba984d1eb41aa5574fa77))


## [0.4.26] - 2026-07-28

### <!-- 0 -->🚀 Features

- add per-entry flow visualization modal ([0229cce](0229cce01bc135d42ceee91592a28d1287e06279))


## [0.4.25] - 2026-07-28

### <!-- 0 -->🚀 Features

- add copy-to-clipboard button for caddy upstream values ([dab57c7](dab57c728824cd05d6ed72beab1116f44d603a99))


## [0.4.24] - 2026-07-28

### <!-- 7 -->⚙️ Miscellaneous Tasks

- add Telegram notifications for releases and CI failures ([3a96b1b](3a96b1b8e0642d4d42985f249b80317e46a678fd))


## [0.4.23] - 2026-07-28

### <!-- 7 -->⚙️ Miscellaneous Tasks

- expand ESLint with React, TypeScript, and JS best-practice rules ([ae5017d](ae5017dc20ff282e0f75d8a16f386c48edaa5dd4))

- add web-lint pre-commit hook for ESLint ([2615f24](2615f24552960e4595043fba2e902964a2dad740))

- add ESLint with custom rule to catch unstable Zustand selectors ([0d982a9](0d982a969c639d047678471325ffaaf39f7755f3))


## [0.4.22] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- infinite render loop from Zustand selectors returning new arrays ([c754998](c754998405e502ef6e56e0bcb0518363fd174001))


## [0.4.21] - 2026-07-28

### <!-- 2 -->🚜 Refactor

- introduce Zustand store and split CSS into per-component files ([492229c](492229c400305b72e234329c108c7fba719e34c3))


## [0.4.20] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- define missing --bg-1 CSS variable causing transparent dropdowns ([0004004](00040042a1edfbb151a68f56994eb036f28056e0))


## [0.4.19] - 2026-07-28

### <!-- 0 -->🚀 Features

- add prune stale entries endpoint and UI ([30f26a3](30f26a36b243dae278c53a1324967834d3bbf586))


## [0.4.18] - 2026-07-28

### <!-- 1 -->🐛 Bug Fixes

- skip wildcard/root domain entries in diagnostics ([86e49dd](86e49dd695ea541b0accb3d8ba894e85a744a3b6))


## [0.4.17] - 2026-07-28

### <!-- 0 -->🚀 Features

- add /api/diagnostics endpoint and Diagnostics tab ([fb072d3](fb072d3a6a7e14d96566d636b1576c10ab1da34c))


## [0.4.16] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- sanitize trailing commas from Caddy host matchers ([d471c1d](d471c1d6dedbcad28de2d46ca12d44fcc5dbb05c))


## [0.4.15] - 2026-07-27

### <!-- 0 -->🚀 Features

- add deploy confirmation modal before applying auth changes ([5e85521](5e85521b2d411f71b76511d20f8ae38443b667ac))


## [0.4.14] - 2026-07-27

### <!-- 0 -->🚀 Features

- add inline auth editing UI with pending changes and edit modal ([92550f8](92550f825c13e39061e36bda2099b41bcfcd2af8))


## [0.4.13] - 2026-07-27

### <!-- 0 -->🚀 Features

- add info tooltips to all auth table column headers ([54f5dc4](54f5dc4fcd2837b46002ff06b64107624c4834bf))


## [0.4.12] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- stop stretching auth table to full container width ([6398e20](6398e2041c2f6c7d11bc4f956d8a96ec57287975))


## [0.4.11] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- normalize auth table row heights ([93c3757](93c3757a7add3861e01d78235f0db7ebf21c0c8b))


## [0.4.10] - 2026-07-27

### <!-- 0 -->🚀 Features

- add search bar, tighten horizontal spacing, clarify N/A ([b42a936](b42a93675fd4866fb534fbe5f4eb9f539d74df87))


## [0.4.9] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- tighten Auth Flows spacing and clarify API Auth column ([5ad8740](5ad8740bd03829959e63b21149d68f785e6e3093))


## [0.4.8] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- use Unlock icon for LAN "None" auth (consistent with WAN) ([461f440](461f4406c4aedbc095d5314289b26f0a20d62ed1))


## [0.4.7] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- change LAN "None" icon from WifiOff to DoorOpen ([0b6676d](0b6676db82e36674cc783b11ca450dccb45ba868))


## [0.4.6] - 2026-07-27

### <!-- 0 -->🚀 Features

- stream dashboard entries loading via SSE with per-service progress ([4ceb1ee](4ceb1ee71a12d37e232c0765d8351f40e039a7b8))


## [0.4.5] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- don't prematurely classify WAN auth before CF Access enrichment ([abd8617](abd8617161edacfdab5872e5b41518637df2da4b))


## [0.4.4] - 2026-07-27

### <!-- 0 -->🚀 Features

- unique icons per auth type in table rows and headers ([04392e6](04392e67d04a8a865da721c37136286becac4722))


## [0.4.3] - 2026-07-27

### <!-- 1 -->🐛 Bug Fixes

- cache config in sessionStorage so version survives page reload ([d1d111c](d1d111c947e4b4e1651f8099cf89b0f2f3aad31a))


## [0.4.2] - 2026-07-27

### <!-- 0 -->🚀 Features

- visual auth legend with icons, color-coded sections, flow diagram ([97dc148](97dc1483ac28bdcd88bac3a2c3dc2504ae9daa42))


## [0.4.1] - 2026-07-27

### <!-- 0 -->🚀 Features

- stream auth inventory via SSE + add auth type legend ([b103a38](b103a38de0282e4f63a152ee3d21d0148dabd614))


### <!-- 1 -->🐛 Bug Fixes

- use content-hashed asset filenames for cache-busting ([4e3849a](4e3849a87c90d1889754f382519fec402fdab70a))


## [0.4.0] - 2026-07-27

### <!-- 0 -->🚀 Features

- add Auth Flows tab with Authentik + CF Access discovery ([099db80](099db80c36dcd516f2f2f932e0f868cb09d27efb))


## [0.2.2] - 2026-07-25

### <!-- 0 -->🚀 Features

- show build timestamp when version is "dev" ([2f629cf](2f629cf91570b9719b4ee25eea3504a9da810f81))


## [0.2.1] - 2026-07-25

### <!-- 2 -->🚜 Refactor

- split Dashboard.tsx, add plan TTL, web API improvements ([4816bc8](4816bc8b7efd0b4b6cecf1336f819011119e65fc))


## [0.2.0] - 2026-07-25

### <!-- 0 -->🚀 Features

- detect and warn on Authentik forward_auth bypass via CF tunnel ([8832f6c](8832f6cdf5cea5202e8e83d238262b4f398ab5fe))

- add OriginServerName support to CF tunnel edit flow ([2b8d6ce](2b8d6ce48b7a9e89b69e2ca953abe5da6ae3b756))

- mark-intentional toggle to suppress warnings per session ([5789d55](5789d557adc79aaa7b21566c8db5e46d90edb69f))

- add long-timeout, forward-auth, compression-no-tls-verify templates ([a8a2551](a8a255191ec458c1c7098c60bd6f7517cad32195))

- show Caddy upstream health ([c9457a5](c9457a500ce14aba2f42bec1db1bcc6d979d832c))

- global activity spinner in sticky topbar ([0fa9a1f](0fa9a1feb2b454aad8a5ed5301a5a203996b4aa6))

- show build version in web UI topbar ([6385e92](6385e923aff551afc4f4cac6942d2d66ece521fb))

- update default Caddyfile template to use import proxy_headers ([56d927a](56d927a2ec18b6bcc1545ee96b74d9c58d7cbcec))

- full status rewrite + CF repair-dns + repair banner ([c8833d7](c8833d7929879d3abf78c5834cac28a79993b0d5))

- add spinner to Cloudflare route buttons while saving ([75457ef](75457efccd61d681f3f7a8dc1c58767f0f4fcef6))

- add drag-to-resize handle on server log panel ([98f99b9](98f99b931c4ce10705e38cec13b80e9d8e0ad68d))

- log load summary with actionable warnings after each data refresh ([a6feb8b](a6feb8b3ff7831da023e4c4f2669ae6ac9f9c7c2))

- surface CF missing CNAME as issue in dashboard ([543a2cc](543a2cc6e469ece060b3dc39553fbeb1ffcd24d7))

- CF DNS CNAME sync, DNS probe with auto-retry, deploy logging ([b69eac1](b69eac13d48bbb79a5402b7c1657e7207bb9361f))

- refine collision detection UI with tooltip badges ([995142c](995142cf1e15215de84427a99fd453e52f51b261))

- merge collision detection + global git banner ([b1e6fa4](b1e6fa440a6a53762cbf34dcdc12939ce977702a))

- add hostname collision decision helper ([7f3a553](7f3a553f1f2cdfc5d90db37e5c2a09f0f85203c2))

- add direct Cloudflare tunnel hosts ([a336c31](a336c3170929dc5a94e0a8ae712766c3328022d9))

- improve sync web and tui status ([ac3f9d7](ac3f9d7bcf3953a230e605c4381d6ea20891560e))

- web tweaks ([a4aef88](a4aef881c703a142416fdac2a2e49175708e2963))

- add cloudflare tunnel sync planning ([736708e](736708e0c8893cf3e56fc79c26d7f167b92379a8))

- refactor web console ([70cf7c1](70cf7c16981a3fe01834283f50d3cb8545aa8b58))

- tighten web app layout ([56c76de](56c76de68fb217f195931dc8d5442c33e902c1af))

- convert web ui to react spa ([04d6d80](04d6d805c8564537e191fbcb03bb013e6e41967e))

- add web config tests ([2cabf0e](2cabf0ef4a0dd39180f51d1ac229c0b273ca687c))

- clarify web loading progress ([277d789](277d789e808b55ddf3b9bcb4943dd7411d81c4e6))

- refine web dashboard UI ([f7ddccb](f7ddccb4b9d11c6a348dbadb0668e8a2c595b7ec))

- support web config editing ([3abb78e](3abb78ed86f14748648f58b1331747d94441e326))

- show sanitized web config summary ([ba42dad](ba42dad1609513b664a4d2e13e02a2b5e7048b7b))

- back web sync controls with apply API ([b0d0ade](b0d0ade0fe8c2f092c2daf43b08cec4c1f42730c))

- improve web dashboard usability ([bf3909a](bf3909a02514fa497a1646f58fe7be669304ce0f))

- build browser UI workflow ([038d1a7](038d1a7a43db1a2e0eb974f819516e1634632449))

- add shared runtime and web UI foundation ([4f3b9df](4f3b9df97584f43c2ef4587f142c7466456a30e9))

- implement CloudflareClient write methods (Step 2) ([7bd926d](7bd926d5de08c233e455978b975fa12d4e4cce8e))

- Cloudflare tunnel config with interactive wizard and TUI improvements ([69b3919](69b391960e710f5c6fcfc2de8f89b7534a638e94))

- bunch-o-changes - tui is mostly working ([0efa80d](0efa80dbebb978848546ddb157a0a7e45912484b))


### <!-- 1 -->🐛 Bug Fixes

- query Unbound DNS directly instead of system resolver ([aa33a0a](aa33a0a5705f1a65acb24f55e091ffc3e87acacf))

- replace conflicting A/AAAA/CNAME records instead of erroring ([7cd802e](7cd802e5074e87453442dd21e32dea552adad350))

- metric card tones now reflect zero-state correctly ([8f4c108](8f4c108e48865d1fe7e3df1b3fc3b15485948ce9))

- detect conflicting DNS records before CNAME creation in EnsureDNSRecord ([dc9b4c5](dc9b4c5b20bbf30fcab8ae5e467a5b1226a81f88))

- surface DNS CNAME failure as warning instead of silent success ([7aff73a](7aff73ae9bc2ccb0428aa1da923605f99588dd84))

- delete companion matchers (e.g. @name_external) on entry removal ([ceac4dd](ceac4dd32b05814b6a53a7106ce09fcf2cfdc536))

- wire Params through preview/UI and fix forward-auth parser ([4ee2362](4ee23622b4f13d0efc527cde36006262cd0f2cd2))

- show web sync controls ([58c56ec](58c56ece706c62da3544f0892da0a1f88f32fa65))


### <!-- 10 -->💼 Other

- redesign Modify modal, fix CF :80 stripping, add query cmd + exclude-hostnames ([8712942](871294257d515262eab8ae797452a189ebc10595))


### <!-- 3 -->📚 Documentation

- update plan.md with completed items and current status ([7eebdd8](7eebdd8185632050671ef1310995f50c742b3e71))


### <!-- 5 -->🎨 Styling

- polish web app density ([2dfc1f3](2dfc1f3881e243da6a92498fe727a9e294323da9))


## [0.1.0] - 2025-09-15

### <!-- 0 -->🚀 Features

- adguard sync workign ([820613c](820613c0c93fb3115a690a775cdb48be41cdd621))


### <!-- 1 -->🐛 Bug Fixes

- version template error ([5407f14](5407f1499617cb7dac03eadaa1aa0382558d2248))


### <!-- 3 -->📚 Documentation

- update readme ([f21a1be](f21a1be82c74e7cfa563b3fcd2643f0d7e948938))


## [0.0.1] - 2025-05-06

### <!-- 0 -->🚀 Features

- initial commit ([626cf14](626cf14a3b96da7fb32b081c2529c6fc502bcdc4))


<!-- generated by git-cliff -->
