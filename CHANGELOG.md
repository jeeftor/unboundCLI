# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### <!-- 0 -->🚀 Features

- selective prune with checkboxes + Prune All button ([dffaa85](dffaa85b59fbaa153de336093e7ef83f9af4aae2))


### <!-- 1 -->🐛 Bug Fixes

- remove dead Sidebar.css, memoize StatusChip, add ErrorBoundary to lazy route ([30e62ac](30e62ac3cae10b94b97fc300e1f4915e3c9840a5))

- modal accessibility, remove dead Sidebar, log Go write errors ([ef623f0](ef623f023241df67116a413e7668ae526cbee8e1))

- Go backend bugs, dead code removal, bundle splitting, CI/CD updates ([28eb015](28eb0155e7a7b768689f4a91fcd3351d2a575fca))

- resolve all 138 eslint warnings ([5c90c75](5c90c7587799c9bda5b808f6961f84bdd8364e72))


### <!-- 2 -->🚜 Refactor

- extract reusable components, fix bugs, consolidate CSS ([aad4029](aad4029b7b7a7c6d7c511f1bff17b812af117525))


### <!-- 3 -->📚 Documentation

- update changelog ([03f4c35](03f4c356c60b65022d600e231e89ca375cdca1fa))

- update changelog ([f8aa6c2](f8aa6c25db6f14dceccd8e1223fa969828f53de6))

- update changelog ([f298776](f29877602668377c5900077c3a6e428320ada0c0))

- update changelog ([b15dc1b](b15dc1b91753ff4b480bad23be3047c5d2b6f232))

- update changelog ([b2f169d](b2f169de8ffe96c2279189f29faca5e4bac1f609))

- update changelog ([571f7ad](571f7ad5ba71812db8f6dd363f0846d2fceffd6d))

- update changelog ([fe1dfe2](fe1dfe26f25da89b8570d40f9afae37ec43e064e))

- update changelog ([80ea0c4](80ea0c4202be50a7e897ef3eb23f95336ba46997))

- update changelog ([cb5d554](cb5d554a9d0a41dced290317e02b980855e2c805))


### <!-- 4 -->⚡ Performance

- memoize remaining list-rendered components ([bfa0a80](bfa0a8085724404ed5d2f3b203d275ca376db018))


### <!-- 7 -->⚙️ Miscellaneous Tasks

- migrate to @eslint-react/eslint-plugin for ESLint 10 support ([a1daf5c](a1daf5c752de9bfde3416bc7b9d4d2e12ca3f684))


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
