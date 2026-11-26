# 网易云音乐插件

配置测试市场地址后，从 Yoho Music 的「插件市场」安装或更新 **网易云音乐**。点「登录」，再用网易云音乐 App 扫描 Yoho 弹窗中的二维码并在手机上确认。插件查询账号成功后将会话存入 NSGo Secrets。退出登录会删除此插件的本地会话。

登录后可以搜索歌曲、专辑、艺人、歌单和播客，浏览每日推荐、排行榜和音乐云盘，查询最近播放和收藏，编辑自建歌单，收听私人 FM，浏览播客和评论。在线播放、音质、账号权限和地区可用性由网易云返回。歌词支持 LRC、翻译和逐字同步。

发现页提供公开推荐歌单和排行榜。登录后还能读取最近一周与所有时间的听歌排行、每日推荐、私人雷达、个性化歌单和云盘。私人漫游提供默认推荐、熟悉的歌和新发现三种模式，播放队列消耗后继续取歌。私人漫游依赖网易云音乐 App 中的「个性化服务」设置。

心动模式从当前账号的「我喜欢的音乐」选择可播放的种子歌曲，通过网易云心动接口持续补歌。点击发现页或「为你推荐」中的「心动模式」卡片即可开始收听。插件会过滤近期重复和不可播放的歌曲，并按登录会话隔离推荐进度。喜欢歌单为空、推荐为空或请求失败时会显示错误，只有心动接口成功返回可播放的新推荐才会开始或继续供歌。

扫码流程和功能以 [sogud/mygo-music-netease](https://github.com/sogud/mygo-music-netease/tree/82696baa3f773d7661a568a430bde8d68f8d0f43)、[AkinoKaede/mygo-music-netease-cloud-music](https://github.com/AkinoKaede/mygo-music-netease-cloud-music) 和 [hubandad/mygo-music-netease](https://github.com/hubandad/mygo-music-netease) 为参考。宿主的 AES 编码函数复用了 MyGo Music v0.1.25 的实现，沿用 AGPL-3.0。

心动请求与连续补歌逻辑参考 sogud 的 `82696baa3f773d7661a568a430bde8d68f8d0f43` 提交，以及 AkinoKaede 的 `b9bf374cce816e87ed3ca58e293ff4001842c1cb` 提交中 `src/radio.ts` 的实现。相关许可与版权说明保留在 `heart.js` 文件头。

## 重新打包

修改脚本后递增 `nsgo.yaml` 中的版本和 `build_number`。从项目根目录使用既有签名根授权的发布证书和 `NSGO_RELEASE_KEY` 重新打包。签名根的公钥位于此目录的 `root.pub`。

```sh
go tool nsgo publish -cert "$YOHO_NETEASE_RELEASE_CERT" -entry index.js \
  -out internal/plugin/bundled/netease/netease.nsgopkg internal/plugin/bundled/netease
```

重新打包后同步更新仓库根目录 `plugin-market.json` 中的版本号和构建号。测试时一起提供清单和签名包。Yoho 会在用户点击「安装」或「更新」后下载插件包。

`TestNetEaseQRLoginAndMedia` 使用替代传输在 NSGo 运行时测试二维码回执、会话保存和恢复、搜索、歌单创建、歌词及播放权限。
