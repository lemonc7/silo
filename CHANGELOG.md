# Changelog

## [0.3.0](https://github.com/lemonc7/silo/compare/v0.2.1...v0.3.0) (2026-05-21)


### Features

* 实现获取详情页链接的逻辑 ([697e596](https://github.com/lemonc7/silo/commit/697e596a958d474db78e59cf4c5ed0d0c4538a0a))


### Bug Fixes

* 修复资源详情页代码逻辑，增加空资源处理逻辑 ([14113f7](https://github.com/lemonc7/silo/commit/14113f713d15451c1bd7e8775acc4f75c21c3773))
* 增加详情页链接爬取逻辑 ([21059a1](https://github.com/lemonc7/silo/commit/21059a11fe4c403e9b5443253cc3ed3de84b9d64))
* 调整包名，调整详情页链接获取逻辑 ([2a6719b](https://github.com/lemonc7/silo/commit/2a6719bf76757802b05a98be573839b9d73796d3))

## [0.2.1](https://github.com/lemonc7/silo/compare/v0.2.0...v0.2.1) (2026-05-19)


### Bug Fixes

* 修复资源站登录bug，调整cookies和校验逻辑 ([d4420ca](https://github.com/lemonc7/silo/commit/d4420ca4a6f0d870e9372b684cda2842d0c7fb48))
* 定义resource接口，重构Login方法为EnsureSession，增加context控制 ([f4fd496](https://github.com/lemonc7/silo/commit/f4fd4960ed1d77996c4051883e492a6ea0ceab60))
* 定义Search接口，准备完善搜索磁力链接的功能 ([089d3d0](https://github.com/lemonc7/silo/commit/089d3d06bde4e7ae771b2b0e0fe9f9dee54d809f))
* 调整browser参数设置，区分测试和生产环境 ([ca333a6](https://github.com/lemonc7/silo/commit/ca333a6355b57e9e6ed072be6e5022b7a6b22692))

## [0.2.0](https://github.com/lemonc7/silo/compare/v0.1.2...v0.2.0) (2026-05-18)


### Features

* 增加qb下载磁力链接的逻辑 ([1d57588](https://github.com/lemonc7/silo/commit/1d575881a8cc8b8ead0518df45a48c96d25a3ad9))


### Bug Fixes

* 优化cookie注入过程 ([964343b](https://github.com/lemonc7/silo/commit/964343be334e394715f8fd38ea2a80445b1ade43))
* 优化登录资源站逻辑，修复关闭弹窗bug ([95c2a48](https://github.com/lemonc7/silo/commit/95c2a48ad4fd85d04cd92b092857b5882ee40eb1))
* 修复 cookie bug，提取 page 的 cookie，而不是browser ([b8e3347](https://github.com/lemonc7/silo/commit/b8e3347ea5d8a8dab122e12029d1642a41a6e769))
* 调整爬取TMDB数据的处理方式 ([acccbf7](https://github.com/lemonc7/silo/commit/acccbf774a33975fa6a7cb78ca64a22b72f56343))
* 调整表格设计 ([88dd056](https://github.com/lemonc7/silo/commit/88dd056469e0e3f48fbcde92e799e5e0b6025f47))

## [0.1.2](https://github.com/lemonc7/silo/compare/v0.1.1...v0.1.2) (2026-05-17)


### Bug Fixes

* 调整资源站代码，先删除其他代码，编写对应的登录逻辑 ([e25f3ae](https://github.com/lemonc7/silo/commit/e25f3ae532bb7d0ec969af17ecafbc035568fb70))

## [0.1.1](https://github.com/lemonc7/silo/compare/v0.1.0...v0.1.1) (2026-05-17)


### Bug Fixes

* 测试自动发版 ([dcbb526](https://github.com/lemonc7/silo/commit/dcbb5266639f2a3820d98888862122d1faaad0f1))
