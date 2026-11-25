---
version: alpha
name: "Ian's Workspace"
description: "伊恩个人主页（ianneo.xyz）的设计规范：一张 550px 宽的白纸，同字号的墨色首句与灰色正文，方角小标签，手写体落款。含深色主题。提取自 ianneo.xyz（2026-11-25）。"
---

<!-- 提取自线上站点。令牌是站点样式表里的作者声明值；尺寸、对比度与断点行为来自浏览器实测。深色令牌以 -dark 结尾。 -->

## Overview

把它想成一封打好的信，放在一张白纸上：没有导航，没有卡片，没有图。一段墨色的开场白，下面几段铅笔灰的正文，一个深色的小按钮，最后用虚线撕开，落一个手写签名。

- **是什么**：产品设计师伊恩的个人主页，只有一页。用五段短文字交代自己正在做的事，每个项目名嵌在句子里，可以点开。
- **给谁看**：从 Dribbble、GitHub、X 或别人的推荐点过来的人，要在十秒内判断"这个人是谁、在做什么、怎么联系"。
- **应有的感受**：干净、平实、有一点纸感。不推销，不炫技。
- **尺度感**：全站只有一种正文字号 15px、一种行高 28px。开场白和正文同号，只靠墨色与灰色分开。
- **结构感**：单栏，一屏放得下，整块内容垂直居中略偏上。

## Colors

```yaml
colors:
  primary: "#09090B"
  button: "#27272A"
  button-ink: "#FFFFFF"
  button-hover: "#3F3F46"
  neutral: "#FAFAF9"
  surface: "#FFFFFF"
  text-body: "#71717A"
  line: "#E4E4E7"
  marker: "#FFD43B"
  marker-ink: "#09090B"
  primary-dark: "#F4F4F5"
  button-dark: "#EDEDF0"
  on-primary-dark: "#18181B"
  button-hover-dark: "#FFFFFF"
  neutral-dark: "#242427"
  surface-dark: "#151517"
  dialog-surface-dark: "#1B1B1E"
  text-body-dark: "#B0B0B9"
  line-dark: "#343438"
  marker-ink-dark: "#18181B"
```

颜色只做一件事：区分"作者亲口说的"和"其余的"。

- **Primary（#09090B）**：墨色。开场白、项目名、弹窗标题——凡是作者亲口说的、需要被读到的。
- **Text body（#71717A）**：正文灰。其余段落、链接、落款。对白底 4.8:1。
- **Neutral（#FAFAF9）**：项目名标签的底、弹窗里的二维码面板、复制按钮。比白只深一丝，带一点暖。
- **Surface（#FFFFFF）**：纸。
- **Line（#E4E4E7）**：链接的下划线、页脚的虚线，也是标签悬停时的底色。
- **Button（#27272A）、Button hover（#3F3F46）、Button ink（#FFFFFF）**：全站唯一的按钮，深灰底白字。
- **Marker（#FFD43B）与 Marker ink**：荧光笔黄，用于手绘的高亮笔触。它是模板的一部分，首页没有出现。
- 没有品牌色，没有链接蓝，没有错误色。

深色主题把纸换成近黑（#151517），字换成浅灰，按钮反相成浅底深字（#EDEDF0 配 #18181B），弹窗用略浅一档的 #1B1B1E，二维码保持白底以保证可扫。

## Typography

```yaml
typography:
  body-md:
    fontFamily: "'SN Pro', -apple-system, BlinkMacSystemFont, 'PingFang SC', sans-serif"
    fontSize: 15px
    fontWeight: 400
    lineHeight: 28px
  heading-md:
    fontFamily: "'SN Pro', -apple-system, BlinkMacSystemFont, 'PingFang SC', sans-serif"
    fontSize: 15px
    fontWeight: 500
    lineHeight: 28px
  brand:
    fontFamily: "'SN Pro', -apple-system, BlinkMacSystemFont, 'PingFang SC', sans-serif"
    fontSize: 16px
    fontWeight: 500
    lineHeight: 28px
  label-md:
    fontFamily: "'SN Pro', -apple-system, BlinkMacSystemFont, 'PingFang SC', sans-serif"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 22px
  label-sm:
    fontFamily: "'SN Pro', -apple-system, BlinkMacSystemFont, 'PingFang SC', sans-serif"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 22px
  label-xs:
    fontFamily: "'SN Pro', -apple-system, BlinkMacSystemFont, 'PingFang SC', sans-serif"
    fontSize: 12px
    fontWeight: 400
    lineHeight: 20px
  sign-off:
    fontFamily: "'Homemade Apple', cursive"
    fontSize: 15px
    fontWeight: 400
    lineHeight: 26px
```

- **一款字体管到底**：自托管的可变字体 SN Pro 管拉丁字母与数字；汉字落到系统字体，在 Apple 设备上是 PingFang SC（实测）。完整字体栈以苹果系统字体和微软雅黑兜底。
- **开场白就是正文**：首句与后续段落同为 15px / 28px、400 字重，唯一的区别是墨色对灰色。不要为"主标题"另设字号或字重。
- **行高宽绰**：28px 配 15px 字号，约 1.87 倍。段与段之间也隔 28px——恰好一行的高度，所以整页读起来是一段段"隔一行的短文"。
- **品牌名略大一号**（16px、500），是页面上唯一超过 15px 的界面文字。
- **标签与按钮用 14px**；弹窗里的辅助文字用 13px 与 12px。
- 没有字距调整，没有全大写，没有斜体，没有加粗强调。
- **落款 "Cheers, Ian" 用手写体 Homemade Apple**（15px / 26px），全站唯一的手写体。
- 模板里的 h1、h2 默认是 15px / 500：即便在长页面，标题也只比正文多半个字重。

## Layout & Spacing

```yaml
spacing:
  xs: 2px
  sm: 4px
  md: 8px
  lg: 16px
  xl: 24px
  2xl: 28px
  3xl: 40px
  4xl: 48px
  5xl: 56px
  sheet-max-width: 598px
  measure: 550px
  gutter: 24px
```

- **一张纸**：内容页最大宽度 598px 居中，左右各 24px 边距，正文行宽 550px——15px 字号下约 36 个汉字一行。
- **垂直位置**：首页内容不满一屏，整块内容在视口里浮动。多余的高度按 2 : 3 分给上方和下方，重心略高于几何中心，因为真居中会显得偏低。上方至少 64px，下方至少 96px。
- **纵向节奏**：页头（40px 高）→ 48px → 正文 → 40px → 按钮 → 56px → 页脚（上内边距 24px）。正文各段之间 28px。
- **页头**：24px 圆形头像、8px、品牌名。主题切换按钮贴右边缘，点击区 44×44（图标 17px）。
- **页脚**：一行。左边是钢笔图标加手写签名，右边是五个图标链接；上方一条虚线。
- **手机（宽度小于 640px）**：回到文档流，上边距 64px；页脚改为纵向左对齐，签名在上、图标在下。整页可以滚动。
- **长页面（模板）**：纸的内边距为上 80、下 180px；底部 160px 是逐层加深的背景模糊，正中浮着一个"来找我聊聊"按钮，距底 48px。
- 页面平滑滚动，滚动条槽位常驻，不因滚动条出现而抖动。

## Elevation & Depth

完全平面：没有投影，没有层级。区分只靠两级底色——白色的纸，和比纸深一丝的灰瓦片（标签、二维码面板）。

- **唯一抬起来的是联系弹窗**：白底、直角、无描边，投影 0 12px 32px 18% 黑加 0 2px 8px 6% 黑。背后的页面退后：蒙一层 25% 的纸色并模糊 4px。
- **长页面的底部**用渐进式背景模糊把页脚压在内容下方；浮动按钮带 0 8px 24px 18% 黑的投影。
- 深色主题下弹窗的投影加深（最高 45% 黑），落款的钢笔图标改为滤色混合以保持可见。

## Shapes

```yaml
rounded:
  none: 0px
  xs: 2px
  sm: 6px
  full: 9999px
```

- **近乎直角**。按钮与图标按钮 2px；标签、复制按钮、分段选项、弹窗全部 0。
- 圆角在这套设计里不承担表达：只有头像是正圆，社交图标的点击区是 6px。
- 荧光笔笔触是手绘的 SVG 路径，不规则的边缘是刻意的。

## Motion & Micro-interaction

```yaml
motion:
  state-change:
    duration: 150ms
    delay: 0ms
    timingFunction: ease
  icon-swap:
    duration: 140ms
    delay: 0ms
    timingFunction: ease
  backdrop:
    duration: 180ms
    delay: 0ms
    timingFunction: ease
  sign-pen:
    duration: 300ms
    delay: 0ms
    timingFunction: cubic-bezier(0.16, 1, 0.3, 1)
  image-zoom:
    duration: 600ms
    delay: 0ms
    timingFunction: cubic-bezier(0.25, 0.1, 0.25, 1)
  floating-cta:
    duration: 500ms
    delay: 0ms
    timingFunction: cubic-bezier(0.76, 0, 0.24, 1)
  marker-draw:
    duration: 700ms
    delay: 0ms
    timingFunction: ease-out
```

几乎不动。悬停只做最小幅度的位移与变色，全部 150ms、默认的 ease 曲线。

- **会动的**：链接的文字与下划线变墨色，右上箭头向右上挪 1px；标签底色变 #E4E4E7；社交图标上浮 2px；按钮底色加深，按下时下沉 1px。
- **不动的**：没有入场动画，没有滚动动效，没有循环播放的装饰。
- **仅有的几处"表情"**：悬停签名时钢笔图标旋转 -6°（300ms）；切换主题时月亮与太阳图标 140ms 交叉淡入；弹窗遮罩 180ms 淡入，关闭时 180ms 淡出。
- **模板组件**：图片框悬停时图片在 600ms 内放大到 1.09；荧光笔笔触画完后文字在 700ms 内变为墨色；长页面的浮动按钮以 500ms 滑入。
- 悬停效果只在支持精确指针的设备上启用；图片缩放还额外要求用户没有偏好减少动效。
- **prefers-reduced-motion**：全部过渡与动画压缩到近乎 0，页面滚动改为即时。

## Voice & Content Strategy

- **语言**：全中文，第一人称。产品名保留英文（Design Engineer Notes、Indie Builders、Topinspo、Portfolio）。
- **语气**：平实、口语、不推销。用"我也是……""最近在写……""还做了……"的流水句式交代经历，像跟熟人聊天。
- **段落**：每段只说一件事，两到四行；段与段之间隔一行。
- **标点**：全角标点；列举用顿号；数字与量词直接相连（200+ 人、300+ 人、一万多个 Star）。
- **行动用语**：唯一的按钮是"来找我聊聊"。弹窗标题同名，选项是"联系方式"下的"微信 / 邮箱"，结束语是"欢迎来打个招呼，一起交流"。不用"提交""发送""联系我们"这类表单用语。
- **落款**：英文手写体的 "Cheers, Ian"，全站唯一的英文句子。
- **无障碍文字**：跳转链接写作"跳到正文"，主题按钮是"切换到暗色模式"。
- **首页没有的东西**：导航、标题层级、按钮组、图片、第二屏。

## Components

```yaml
components:
  page:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text-body}"
    typography: "{typography.body-md}"
  intro:
    textColor: "{colors.primary}"
    typography: "{typography.body-md}"
  brand:
    textColor: "{colors.primary}"
    typography: "{typography.brand}"
  button-primary:
    backgroundColor: "{colors.button}"
    textColor: "{colors.button-ink}"
    typography: "{typography.label-md}"
    rounded: "{rounded.xs}"
    height: 32px
    padding: 2px 12px
  button-primary-hover:
    backgroundColor: "{colors.button-hover}"
  inline-link:
    textColor: "{colors.text-body}"
    typography: "{typography.body-md}"
  inline-link-hover:
    textColor: "{colors.primary}"
  chip:
    backgroundColor: "{colors.neutral}"
    textColor: "{colors.primary}"
    typography: "{typography.label-md}"
    rounded: "{rounded.none}"
    padding: 0px 6px
  chip-hover:
    backgroundColor: "{colors.line}"
  icon-button:
    textColor: "{colors.text-body}"
    rounded: "{rounded.xs}"
    size: 44px
  icon-button-hover:
    backgroundColor: "{colors.neutral}"
    textColor: "{colors.primary}"
  social-link:
    rounded: "{rounded.sm}"
    height: 28px
    width: 24px
  footer-rule:
    backgroundColor: "{colors.line}"
    height: 1px
  sign-off:
    textColor: "{colors.text-body}"
    typography: "{typography.sign-off}"
  dialog:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text-body}"
    typography: "{typography.body-md}"
    rounded: "{rounded.none}"
    padding: 24px
    width: 360px
  dialog-title:
    textColor: "{colors.primary}"
    typography: "{typography.heading-md}"
  segment:
    textColor: "{colors.text-body}"
    typography: "{typography.label-sm}"
    height: 38px
    padding: 5px 8px
  segment-active:
    backgroundColor: "{colors.neutral}"
    textColor: "{colors.primary}"
  copy-id:
    backgroundColor: "{colors.neutral}"
    textColor: "{colors.primary}"
    typography: "{typography.label-md}"
    rounded: "{rounded.none}"
    padding: 1px 7px
  marker-highlight:
    backgroundColor: "{colors.marker}"
    textColor: "{colors.marker-ink}"
    typography: "{typography.body-md}"
  page-dark:
    backgroundColor: "{colors.surface-dark}"
    textColor: "{colors.text-body-dark}"
  intro-dark:
    textColor: "{colors.primary-dark}"
  button-primary-dark:
    backgroundColor: "{colors.button-dark}"
    textColor: "{colors.on-primary-dark}"
  button-primary-dark-hover:
    backgroundColor: "{colors.button-hover-dark}"
  chip-dark:
    backgroundColor: "{colors.neutral-dark}"
    textColor: "{colors.primary-dark}"
  chip-hover-dark:
    backgroundColor: "{colors.line-dark}"
  dialog-dark:
    backgroundColor: "{colors.dialog-surface-dark}"
    textColor: "{colors.text-body-dark}"
  marker-highlight-dark:
    backgroundColor: "{colors.marker}"
    textColor: "{colors.marker-ink-dark}"
```

- **开场白**：一段 15px 的墨色文字，没有任何标题式的视觉强化。
- **行内链接**：文字保持正文灰，下划线是 #E4E4E7 的 1px 细线（偏移 4px），后面跟一个 10px 的右上箭头。悬停时文字与下划线变墨色，箭头向右上挪 1px。
- **项目名标签**：嵌在句子里的 14px 小方块，灰底墨字，方角，高 22px，左右内边距 6px，右侧带箭头；悬停底色变 #E4E4E7。它告诉读者"这是一个产品名"，而不是把项目抽出来列成清单。
- **换行处理**：链接与其后的标点固定在同一行，避免箭头后面跟着一个逗号掉到下一行；全角标点后面的标签间距收紧半个字（font-feature-settings 的 halt）。
- **主按钮**：深灰底白字、2px 圆角、高 32px、文字 14px、右侧 12px 箭头；悬停加深，按下下沉 1px。模板的通用按钮高 40px、文字 15px。
- **图标按钮**：主题切换，44px 点击区、2px 圆角；悬停浮出灰底、图标变墨色。
- **页脚签名**：26px 的钢笔图标加手写签名；悬停签名变墨色，钢笔旋转 -6°。
- **联系弹窗**：原生 dialog，宽 360px、内边距 24px、直角。标题 15px / 500 墨色，右上角 32px 关闭按钮；"微信 / 邮箱"是两个并排的方角选项（选中者灰底墨字，高 38px）；下面是灰底的二维码面板（二维码 200px）与可复制的微信号（14px 的灰底小方块，复制后显示对勾）。手机上弹窗左右各留 24px，二维码缩到 176px。
- **焦点**：2px 正文灰描边，偏移 4px。页首有"跳到正文"的跳转链接。
- **图标**：1.5px 描边的线性图标（月亮、太阳、关闭、邮件、右上箭头）。

## Do's and Don'ts

- Do 保持开场白与正文同号，用墨色对灰色区分。
- Do 把项目名嵌在句子里，做成灰色小标签。
- Do 让内容在首屏垂直居中略偏上（2 : 3）。
- Do 保持一个按钮、一个落款、一条虚线的极简结构。
- Don't 给开场白加大字号或加粗，不要引入第二种标题层级。
- Don't 使用圆角卡片、投影或边框来分组。
- Don't 使用 400 和 500 以外的字重。
- Don't 把手写体用在签名以外的地方。
- Don't 给首页添加导航、横幅图或第二屏——它是一页纸。
