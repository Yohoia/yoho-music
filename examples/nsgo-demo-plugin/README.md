# Yoho Music 的 NSGo 示例插件

示例提供 `search` 和 `stream`，复用 MyGo Music 演示插件的测试曲目和 `tone.mp3`。从 Yoho Music 项目根目录打包。

1. 用 `go tool nsgo keygen` 分别生成测试根密钥和发布密钥。将根公钥以单行文本保存到 Yoho 数据目录的 `nsgo-plugins/roots/demo.pub`。默认数据目录是 `~/Library/Application Support/Yoho Music`，也可以用 `YOHO_MUSIC_DATA_DIR` 启动独立的测试实例。应用启动时读入根公钥。
2. 设置 `NSGO_ROOT_KEY` 和 `NSGO_RELEASE_KEY` 后，在同一个终端为示例签名。`YOHO_DEMO_RELEASE_PUBLIC_KEY` 是发布公钥。

   ```sh
   go tool nsgo certify -key "$YOHO_DEMO_RELEASE_PUBLIC_KEY" -apps dev.yoho.music.demo -out /tmp/yoho-demo.cert
   go tool nsgo publish -cert /tmp/yoho-demo.cert -entry index.js -out /tmp/yoho-demo.nsgopkg examples/nsgo-demo-plugin
   ```

3. 用 `python3 -m http.server 8765 --bind 127.0.0.1 --directory internal/plugin/bundled` 提供测试音频。
4. 在 Yoho 的「设置 → 插件 → 安装 NSGo 插件包」选择 `/tmp/yoho-demo.nsgopkg`，确认网络权限后选中侧栏的 **Yoho NSGo Demo**。搜索 `Song` 并点击一首歌播放。

## Yoho 音乐插件接口

插件使用 `nsgo.yaml` 声明包 ID、版本和权限。`search({ query, kind, offset })` 和 `stream({ id, quality })` 是必需的导出函数。`search` 返回结果对象和可选的分页标记 `more`。曲目至少需要 `id` 和 `name`，`duration` 以毫秒计。Yoho 会绑定曲目来源。`stream` 返回 HTTP(S) `url`、音频 `type` 和可选的 `headers`。音频下载和重定向目标均受 `capabilities.network` 约束。

插件可按需导出 `home`、`playlists`、`playlist`、`lyrics`、`account`、`signIn` 和 `signOut`。函数可同步返回 JSON 或返回 Promise。需要向宿主返回 `signIn`、`setup`、`unavailable` 等业务错误时，在异步函数中抛出带 `code` 的错误。

登录函数接收 `signIn({ requestId })`。声明 `capabilities.host: [yoho.signInPrompt]` 后，可从 `nsgo` 导入 `host` 并调用 `await host.call("yoho.signInPrompt", { requestId, text, qr })`。`text` 与 `qr` 均可省略。请求结束、取消或超时后，宿主会拒绝该请求的后续提示。插件应处理拒绝并停止对应的登录流程。成功后返回含 `id` 和 `name` 的账号对象。`account()` 返回当前账号或 `null`，`signOut()` 清除会话。声明 `capabilities.secrets: true` 后可从 `nsgo` 导入 `Secrets` 保存敏感凭据。
