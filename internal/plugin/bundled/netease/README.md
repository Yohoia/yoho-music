# 网易云音乐插件

配置测试市场地址后，从 Yoho Music 的「插件市场」安装或更新 **网易云音乐**。点「登录」，再用网易云音乐 App 扫描 Yoho 弹窗中的二维码并在手机上确认。插件查询账号成功后将会话存入 NSGo Secrets。退出登录会删除此插件的本地会话。

登录后可以搜索歌曲、专辑、艺人、歌单和播客，浏览每日推荐、排行榜和音乐云盘，查询最近播放和收藏，编辑自建歌单，收听私人 FM，浏览播客和评论。在线播放、音质、账号权限和地区可用性由网易云返回。歌词支持 LRC、翻译和逐字同步。

扫码流程和功能以 [sogud/mygo-music-netease](https://github.com/sogud/mygo-music-netease)、[AkinoKaede/mygo-music-netease-cloud-music](https://github.com/AkinoKaede/mygo-music-netease-cloud-music) 和 [hubandad/mygo-music-netease](https://github.com/hubandad/mygo-music-netease) 为参考。宿主的 AES 编码函数复用了 MyGo Music v0.1.25 的实现，沿用 AGPL-3.0。

## 重新打包

修改脚本后递增 `nsgo.yaml` 中的版本和 `build_number`。从项目根目录使用既有签名根授权的发布证书和 `NSGO_RELEASE_KEY` 重新打包。签名根的公钥位于此目录的 `root.pub`。

```sh
go tool nsgo publish -cert "$YOHO_NETEASE_RELEASE_CERT" -entry index.js \
  -out internal/plugin/bundled/netease/netease.nsgopkg internal/plugin/bundled/netease
```

重新打包后同步更新仓库根目录 `plugin-market.json` 中的版本号和构建号。测试时一起提供清单和签名包。Yoho 会在用户点击「安装」或「更新」后下载插件包。

`TestNetEaseQRLoginAndMedia` 使用替代传输在 NSGo 运行时测试二维码回执、会话保存和恢复、搜索、歌单创建、歌词及播放权限。
