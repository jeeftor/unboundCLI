# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### <!-- 0 -->🚀 Features

- selective prune with checkboxes + Prune All button ([b865e3f](b865e3f4215cc1592d4dc5e0c301150ef60a5673))

- add fix-double-login action to create CF Access bypass ([f43f716](f43f716466f7ededcb57915ee907c9477274430b))

- cosign-sign Go binary, add CODEOWNERS, Scorecard, Go 1.26 ([04372f1](04372f19ca5420fcc4ccc429ab9f996e57763208))

- add SBOM workflow with SPDX + CycloneDX ([0063b92](0063b928f5b617d3ec880dd72ff9da3cc3cabd0a))

- add sb.vookie.net to forward_auth registry ([11188ff](11188ff99744864b4e48577e8d890094dc915ecf))

- add forward_auth registry to detect missing auth regressions ([c20865d](c20865df165ce4f521521c15146e6c78a099ec77))

- graceful shutdown, SSE diagnostics stream, repair-dns progress ([40681b1](40681b15135ea2b5a99f6c70c9aab416043b2e30))

- auto-detect Authentik IdP host to avoid false positive ([704215a](704215a17bade552b275fe2e5f094b00a699b8b8))

- standalone visualize page at /visualize/{hostname} ([ccf08a6](ccf08a64138493393cad50f1cb7e900736599aa2))

- deep link support for visualize modal ([50aa1e9](50aa1e9af43ab4200e150b210954dabb6fe1837c))

- show service name instead of 'Service' in diagram nodes ([c06e1aa](c06e1aa23f165bcb7eb4e4d3373cd794f553785c))

- side-by-side diagram+flow layout, custom arrow edges, step numbers ([20a4d61](20a4d61abe15a8c9521eff1da44b5cec5e3fd267))

- replace CSS flow diagrams with React Flow + ELK auto-layout ([e3a9149](e3a9149f03004e4caa9b5ba758e7c9d92a4567fb))

- add /api/health and /api/version endpoints ([57a448c](57a448c805029312a6377513320fea4bfd94bd88))

- add step-by-step request flow tables to VisualizeModal ([0826402](08264023b023d5076a699e4f4ef9eea237aeca05))

- render auth steps as nodes in the flow diagram ([0f13c2c](0f13c2ca736b4f1f070aa1e8bb3916ffc97ff2c1))

- add auth pattern analysis to VisualizeModal ([ab65230](ab652306afbd6029150d0ac9530e5ebacaf32cc3))

- enrich VisualizeModal with auth inventory data (CF Access + Authentik) ([4707eab](4707eabe4933332deb2a2341a67f21c563054952))

- add per-entry flow visualization modal ([02a19ce](02a19ce50ab3dc597a5de642fbe798998224aba8))

- add copy-to-clipboard button for caddy upstream values ([8497abc](8497abc8a2517554b55c92975d70705c00c842a8))

- add prune stale entries endpoint and UI ([30f26a3](30f26a36b243dae278c53a1324967834d3bbf586))

- add /api/diagnostics endpoint and Diagnostics tab ([fb072d3](fb072d3a6a7e14d96566d636b1576c10ab1da34c))

- add deploy confirmation modal before applying auth changes ([5e85521](5e85521b2d411f71b76511d20f8ae38443b667ac))

- add inline auth editing UI with pending changes and edit modal ([92550f8](92550f825c13e39061e36bda2099b41bcfcd2af8))

- add info tooltips to all auth table column headers ([54f5dc4](54f5dc4fcd2837b46002ff06b64107624c4834bf))

- add search bar, tighten horizontal spacing, clarify N/A ([b42a936](b42a93675fd4866fb534fbe5f4eb9f539d74df87))

- stream dashboard entries loading via SSE with per-service progress ([4ceb1ee](4ceb1ee71a12d37e232c0765d8351f40e039a7b8))

- unique icons per auth type in table rows and headers ([04392e6](04392e67d04a8a865da721c37136286becac4722))

- visual auth legend with icons, color-coded sections, flow diagram ([97dc148](97dc1483ac28bdcd88bac3a2c3dc2504ae9daa42))

- stream auth inventory via SSE + add auth type legend ([b103a38](b103a38de0282e4f63a152ee3d21d0148dabd614))

- add Auth Flows tab with Authentik + CF Access discovery ([099db80](099db80c36dcd516f2f2f932e0f868cb09d27efb))

- show build timestamp when version is "dev" ([2f629cf](2f629cf91570b9719b4ee25eea3504a9da810f81))

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

- adguard sync workign ([820613c](820613c0c93fb3115a690a775cdb48be41cdd621))

- initial commit ([626cf14](626cf14a3b96da7fb32b081c2529c6fc502bcdc4))


### <!-- 1 -->🐛 Bug Fixes

- remove dead Sidebar.css, memoize StatusChip, add ErrorBoundary to lazy route ([3f2c3b4](3f2c3b4ca008b9dc9af64883edf7006ad001ab55))

- modal accessibility, remove dead Sidebar, log Go write errors ([83244f3](83244f3f31bd0d47e89654ee3fa5391d1ba93507))

- Go backend bugs, dead code removal, bundle splitting, CI/CD updates ([469c32b](469c32b07017e54467acdd935d485be1d8a0630e))

- resolve all 138 eslint warnings ([7a420c0](7a420c07733f44850a0a83cbbb226eb112150b3c))

- add --repo flag to gh release upload in cosign-sign ([0fd7737](0fd773700c5904df6efc2eb9b240ef3ad169f655))

- use ./*.glob in release upload to satisfy shellcheck ([4bc50b3](4bc50b38530dd85c525837191d5ffa022c46e546))

- fix evaluated-envs format for SLSA builder ([df27929](df27929968496b393d19b426e2e736447e4a93ee))

- add evaluated-envs to SLSA Go builder ([13643ef](13643ef4b2d9cbf608ce9bcc68c7e36fc729d1dd))

- skip internal/status tests (require OPNSense/Caddy server) ([21d0a94](21d0a94154cdae3260176f8ca54d313461d5a85d))

- install Caddy for caddyeditor tests, skip race test ([1e0336e](1e0336ebf1070e1b7c2f87447ed9adfd3392aa96))

- fix EOF newlines in store.ts and CHANGELOG.md ([1177960](117796099b38b0bf3e05bada51771133c8077c74))

- fix trailing whitespace in built/static files, changelog push perms ([9534caf](9534cafb7a75bf691347ecaf05f499e3172b9d60))

- fix pre-commit formatting issues and changelog workflow ([bed8058](bed80589b4bca86ff9e7df0f2e89bff1428b82df))

- fix cliff.toml for git-cliff v4 compatibility ([14267ca](14267caddcf4f51c499493b6b80bf4df71f9b065))

- fix shellcheck word splitting, upgrade git-cliff-action ([e09227a](e09227aa7971e1fdc4405ceccebedb51f50783d2))

- make trivy non-blocking, fix npm ci peer dep conflict in build/lint ([38c82a6](38c82a62ce755639998688f1f7c1300fa905ebd1))

- replace broken trivy-action, fix govulncheck and npm ci ([f633a04](f633a044499f9f9c712f8f95afe156b3870c3b8a))

- concurrency control, security hardening, and reliability improvements ([c075c94](c075c94f0805c0f0044c12408041ca53fd05b2f4))

- clean up error handling, validation, and minor code quality issues ([df0adec](df0adec2d7bd17e19b269d07d18ab7ca58df2d25))

- critical build, CI, and code quality issues ([13fc604](13fc604e9e42574fe5dab23d180c5d29b27e7101))

- flag WAN-exposed hosts with CF Access bypass-only as critical error ([3e6201b](3e6201b57d25eb22bb56ddc8a17b1989703cf502))

- detect conditional forward_auth to avoid false double-login warning ([4902276](49022767e182a90fc6decffa0468a5dc80c23cb1))

- increase ELK node spacing to reduce cramped layout ([33b0f91](33b0f91a9b5d0ef9bc49292a20688fabb71b8dd8))

- always use vertical layout for consistency ([487d596](487d5964c81ab284a7928a5ab6636c1402d067e9))

- arrows now visible, move IdP label to JWT verified ([6ce5987](6ce59878cb8ecc6bd52ee0cfcafe2138ae19b2a5))

- React Flow diagrams — fitView after layout, vertical for 5+ nodes, arrows ([dc9e4ff](dc9e4fffaec83a473a2455d93818acf74064c681))

- flow diagram arrows, CF Access bypass detection, and node rendering ([c21cc04](c21cc04c5890318bec75cb521d6356ff754a7fbe))

- infinite render loop from Zustand selectors returning new arrays ([c754998](c754998405e502ef6e56e0bcb0518363fd174001))

- define missing --bg-1 CSS variable causing transparent dropdowns ([0004004](00040042a1edfbb151a68f56994eb036f28056e0))

- skip wildcard/root domain entries in diagnostics ([86e49dd](86e49dd695ea541b0accb3d8ba894e85a744a3b6))

- sanitize trailing commas from Caddy host matchers ([d471c1d](d471c1d6dedbcad28de2d46ca12d44fcc5dbb05c))

- stop stretching auth table to full container width ([6398e20](6398e2041c2f6c7d11bc4f956d8a96ec57287975))

- normalize auth table row heights ([93c3757](93c3757a7add3861e01d78235f0db7ebf21c0c8b))

- tighten Auth Flows spacing and clarify API Auth column ([5ad8740](5ad8740bd03829959e63b21149d68f785e6e3093))

- use Unlock icon for LAN "None" auth (consistent with WAN) ([461f440](461f4406c4aedbc095d5314289b26f0a20d62ed1))

- change LAN "None" icon from WifiOff to DoorOpen ([0b6676d](0b6676db82e36674cc783b11ca450dccb45ba868))

- don't prematurely classify WAN auth before CF Access enrichment ([abd8617](abd8617161edacfdab5872e5b41518637df2da4b))

- cache config in sessionStorage so version survives page reload ([d1d111c](d1d111c947e4b4e1651f8099cf89b0f2f3aad31a))

- use content-hashed asset filenames for cache-busting ([4e3849a](4e3849a87c90d1889754f382519fec402fdab70a))

- query Unbound DNS directly instead of system resolver ([aa33a0a](aa33a0a5705f1a65acb24f55e091ffc3e87acacf))

- replace conflicting A/AAAA/CNAME records instead of erroring ([7cd802e](7cd802e5074e87453442dd21e32dea552adad350))

- metric card tones now reflect zero-state correctly ([8f4c108](8f4c108e48865d1fe7e3df1b3fc3b15485948ce9))

- detect conflicting DNS records before CNAME creation in EnsureDNSRecord ([dc9b4c5](dc9b4c5b20bbf30fcab8ae5e467a5b1226a81f88))

- surface DNS CNAME failure as warning instead of silent success ([7aff73a](7aff73ae9bc2ccb0428aa1da923605f99588dd84))

- delete companion matchers (e.g. @name_external) on entry removal ([ceac4dd](ceac4dd32b05814b6a53a7106ce09fcf2cfdc536))

- wire Params through preview/UI and fix forward-auth parser ([4ee2362](4ee23622b4f13d0efc527cde36006262cd0f2cd2))

- show web sync controls ([58c56ec](58c56ece706c62da3544f0892da0a1f88f32fa65))

- version template error ([5407f14](5407f1499617cb7dac03eadaa1aa0382558d2248))


### <!-- 10 -->💼 Other

- replace bottom tables with pill badges ([0827105](082710569fe3e74dc26ca1d243be8cdbf4abbe35))

- redesign Modify modal, fix CF :80 stripping, add query cmd + exclude-hostnames ([8712942](871294257d515262eab8ae797452a189ebc10595))


### <!-- 2 -->🚜 Refactor

- extract reusable components, fix bugs, consolidate CSS ([299be60](299be606d76a2b57165cbec3e6daa17f465da919))

- improve error handling, validation, and code organization ([bd047f8](bd047f8db27bed5b2a3538bbe84730d053979f1e))

- split monolithic server.go into domain-specific files ([8fad502](8fad502d6c429cf7602e09c76a7790ff9e1890c3))

- deprecate conditional forward_auth (Pattern E) ([3935846](3935846084fea2e32cc63f56c022c5fb35ed35b8))

- replace flow table with vertical step list ([99640d4](99640d4f9b19dc3b79653116294207e54c6a4614))

- group WAN diagram+flow together, LAN diagram+flow together ([1d71baa](1d71baab9d1925c20bdd60f9774611b021b80032))

- replace status tile grids with config tables ([fbf725d](fbf725d920df2502006563b81f25f116a7174977))

- introduce Zustand store and split CSS into per-component files ([492229c](492229c400305b72e234329c108c7fba719e34c3))

- split Dashboard.tsx, add plan TTL, web API improvements ([4816bc8](4816bc8b7efd0b4b6cecf1336f819011119e65fc))


### <!-- 3 -->📚 Documentation

- update changelog ([a7d4ceb](a7d4ceb6ab756ace610c89ba5253868913b109e1))

- update changelog ([63e23b5](63e23b5a2036f3bddbe5a6952b290e1f396f69df))

- update changelog ([b557931](b5579316390d6cf70d6ada167f9e3eaab154256a))

- update changelog ([8ec2394](8ec23943a1e9d76ed4e8e092ca02ea475d95e414))

- update changelog ([3788114](378811407f3360fad836ce1a9240ea5bf9e76394))

- update changelog ([00c72d3](00c72d3a724ca184f5899269b92eccedc3e5b0f1))

- update changelog ([6e6f94d](6e6f94dbc2a909a5b1b20fff4cb6efd48753c76b))

- update changelog ([ff68a5b](ff68a5b22ed422813b7d04078a9b5486d2dedd47))

- update changelog ([932ef9c](932ef9cf5ca14cafd8ea5d09650f519d6456ae52))

- update changelog ([67e531f](67e531f1ea90b4002297d50b1040555b46ef712d))

- update changelog ([b851678](b8516786ce937535a8eb827b017e488e38925298))

- update changelog ([d8f8edc](d8f8edc148ef8049f461973bd338a91f0d265a36))

- update changelog ([2d7fc17](2d7fc174e181efe688353103e2439300ce746a62))

- update changelog ([58f332a](58f332a9a6e86c66869f1bd109a804477b9bba0f))

- update changelog ([887af92](887af920cb78d35750b04f3ae771292b9980214a))

- update changelog ([57fd942](57fd94250f92d49fd00d3bbb00ae1308cd5e2bb2))

- update changelog ([ee29907](ee29907f05854a3d71e802712d93f47bcd475a1a))

- update changelog ([6d3e1db](6d3e1dba09b58bf4e163e439acd5591e5cfeed2e))

- update plan.md with completed items and current status ([7eebdd8](7eebdd8185632050671ef1310995f50c742b3e71))

- update readme ([f21a1be](f21a1be82c74e7cfa563b3fcd2643f0d7e948938))


### <!-- 4 -->⚡ Performance

- memoize remaining list-rendered components ([145affa](145affadd97e1d4f5c8b487a225d65116c487b58))

- add context propagation, panic recovery, and entries caching ([229f893](229f89390206689f22870694135911f00cc89a9f))

- cache auth inventory on backend, populate at startup ([8c6a7d0](8c6a7d03af83310c98e4335b30df65e520a658ed))

- cache auth inventory in Zustand store, fetch once at startup ([a7596e1](a7596e1a77b1c7597e0201b98f0ea1b2cb30bb8c))


### <!-- 5 -->🎨 Styling

- white circle badges with blue outline and blue number ([7666234](766623400c8451025bc4d740aef8d2b11691c6ae))

- widen VisualizeModal from 720px to 1100px ([cf10655](cf1065571944e8e710957c05d43a4c59a6b53f53))

- polish web app density ([2dfc1f3](2dfc1f3881e243da6a92498fe727a9e294323da9))


### <!-- 7 -->⚙️ Miscellaneous Tasks

- migrate to @eslint-react/eslint-plugin for ESLint 10 support ([be1498c](be1498c4cc8b3e36928df01c22f4f4159da06666))

- fix branch refs, add badges, CodeRabbit, fix Build/Lint ([b05e8af](b05e8af4e4221f9cdf50fc552859791422d86293))

- replace Dependabot with Renovate ([fd94096](fd94096d056585d3ff976869966448625ca39aaf))

- add CVE scanning, Dependabot, SECURITY.md, enable gosec ([7266333](726633368fb8189c4e17db7da7c7856b23bc3223))

- remove temporary Playwright review scripts ([f4cef0e](f4cef0e29a448b3e475ce302dc590889e5dacdf6))

- add Telegram notifications for releases and CI failures ([3a96b1b](3a96b1b8e0642d4d42985f249b80317e46a678fd))

- expand ESLint with React, TypeScript, and JS best-practice rules ([ae5017d](ae5017dc20ff282e0f75d8a16f386c48edaa5dd4))

- add web-lint pre-commit hook for ESLint ([2615f24](2615f24552960e4595043fba2e902964a2dad740))

- add ESLint with custom rule to catch unstable Zustand selectors ([0d982a9](0d982a969c639d047678471325ffaaf39f7755f3))


<!-- generated by git-cliff -->
