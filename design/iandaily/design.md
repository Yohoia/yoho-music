---
version: alpha
name: "Ian's Daily"
description: "伊恩日刊（iandaily.xyz）的设计规范：浅灰画布上浮着一扇白色圆角面板，胶囊控件与悬浮筛选条，配手写体刊名与可拖拽的贴纸标签。含完整的深色主题。提取自 iandaily.xyz（2026-11-25）。"
---

<!-- 提取自线上站点。令牌是站点样式表里的作者声明值；尺寸、对比度与断点行为来自浏览器实测。深色令牌以 -dark 结尾。 -->

## Overview

把它想成 macOS 桌面上浮着的一扇系统窗口：浅灰的桌面，一整块白色大圆角面板，面板里是可以随意排布的内容卡片；控件都是系统级的——胶囊、分段开关、圆形图标按钮、底部悬浮的筛选条。"窗口"克制而精致，热闹的是窗口里的内容。

- **是什么**：伊恩每天更新的中文日报，收录 AI 资讯、一人公司、产品设计、审美提升、产品营销、AI 协作六个分类的链接；每条配封面图、一句编辑判断和来源域名。
- **给谁看**：关注 AI 与产品设计的中文读者，每天来快速扫一遍"今天发生了什么"。
- **应有的感受**：干净、轻、有系统级的精致；同时带一点手作的温度——手写体刊名，以及一堆散落着、可以随手拨动的贴纸标签。
- **尺度感**：正文字号集中在 13–14px，最大的内容标题也只有 15px；唯一的"大字"是 28px 的手写体刊名。
- **颜色**：几乎无彩。色彩由封面图提供，界面是墨、灰、白三种角色，加深浅两套主题。

## Colors

```yaml
colors:
  primary: "#1D1D1F"
  on-primary: "#FFFFFF"
  neutral: "#F5F5F7"
  surface: "#FFFFFF"
  text-muted: "#6E6E73"
  text-subtle: "#86868B"
  line: "#EBEBEB"
  line-strong: "#D2D2D7"
  line-hover: "#D9D9D9"
  hover-tint: "#1516180F"
  error: "oklch(63.7% 0.237 25.331)"
  primary-dark: "#F5F5F7"
  on-primary-dark: "#141416"
  neutral-dark: "#0C0C0E"
  surface-dark: "#141416"
  surface-selected-dark: "#1F1F23"
  surface-raised-dark: "#242427"
  bar-dark: "#18181B"
  bar-line-dark: "#27272A"
  sticker-dark: "#1E1E22"
  sticker-ink-dark: "#C4C4CC"
  text-muted-dark: "#98989F"
  text-subtle-dark: "#71717A"
  line-dark: "#242428"
  line-strong-dark: "#38383E"
  line-hover-dark: "#323238"
  hover-tint-dark: "#FFFFFF14"
```

浅色主题是一组与 Apple 官网相同的灰值，没有一个品牌色。

- **Primary（#1D1D1F）**：墨色，近乎纯黑。所有标题与正文，也是"选中"的颜色——选中的筛选胶囊、推广按钮用它做底色配白字。
- **Text muted（#6E6E73）**：摘要、说明、未选中项。对白底 5.1:1。
- **Text subtle（#86868B）**：引导行（分类 · 来源）、日期、计数。对白底 3.6:1，达不到正文级的对比度，所以只用在 11–13px 的辅助信息上，从不承载需要通读的内容。
- **Neutral（#F5F5F7）**：画布、骨架屏、"今日看点"卡、分段开关的底。比白深一档，表示"垫在下面的一层"。
- **Surface（#FFFFFF）**：主面板、内容卡片、悬浮筛选条。
- **Line（#EBEBEB）、Line strong（#D2D2D7）、Line hover（#D9D9D9）**：三档线色。卡片描边用最浅的一档，悬停加深到 line-hover；虚线分隔与未选中的指示点用最深的一档。
- **Hover tint**：悬停底色，#151618 的 6%，半透明。
- **Error**：仅用于个别红色文字（如错误提示），首页瀑布流里不出现。

深色主题不是简单反相，而是一组偏冷的锌灰：

- **底色由近黑到深灰分成几级**：画布 #0C0C0E，面板与卡片 #141416，悬浮筛选条 #18181B（描边 #27272A），贴纸标签 #1E1E22，选中的导航项 #1F1F23，浮起的控件 #242427。级差很小，靠 1px 线而不是色块来分界。
- **文字**：主文字 #F5F5F7，次级 #98989F，辅助 #71717A。
- **线**：#242428、#38383E、#323238。
- **反相**：选中的筛选胶囊、推广按钮在深色下变成浅底深字（#F5F5F7 配 #141416）。"选中"始终是屏幕上对比最强的那一块。
- 主题由侧栏底部的圆形按钮切换，状态写在 html 的 data-theme 属性上。

## Typography

```yaml
typography:
  wordmark:
    fontFamily: "'Biro Script', cursive"
    fontSize: 28px
    fontWeight: 400
    lineHeight: 1.2
    letterSpacing: -0.03em
  section-title:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 15px
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: -0.02em
  card-title:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 14px
    fontWeight: 500
    lineHeight: 1.4
    letterSpacing: -0.01em
  list-heading:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 13px
    fontWeight: 600
    lineHeight: 1.5
    letterSpacing: -0.01em
  body-md:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 16px
    fontWeight: 400
    lineHeight: 1.5
  body-sm:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.55
  bio:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 2
  label-md:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1
  label-md-medium:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 13px
    fontWeight: 500
    lineHeight: 1
  label-sm:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 12px
    fontWeight: 400
    lineHeight: 1
  label-sm-medium:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 12px
    fontWeight: 500
    lineHeight: 1
  kicker:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 11px
    fontWeight: 500
    lineHeight: 1.2
    letterSpacing: 0.01em
  sticker:
    fontFamily: "'Inter', 'PingFang SC', 'Noto Sans SC', sans-serif"
    fontSize: 11px
    fontWeight: 400
    lineHeight: 1.3
```

- **拉丁字母与数字用 Inter**（自托管的可变字体），**汉字落到系统字体**——在 Apple 设备上是 PingFang SC（实测）。站点同时从 Google Fonts 加载了 Noto Sans SC，但它排在系统字体之后，只在没有苹方的平台上接管。
- **完整的字体栈顺序有意义**：Inter → 苹方 / SF Pro → Noto Sans SC → 微软雅黑 → system-ui。中英文混排时，同一个词里英文取 Inter、汉字取苹方，这是预期行为。
- **刊名用 Biro Script 手写体**，28px，是全站唯一的展示字体。它是"刊头"，不是标题样式，不要用在别处。
- **标题很小**：卡片标题 14px、栏目标题 15px，都是 500 字重，带 -0.01em 到 -0.02em 的字距。没有更大的标题层级。
- **行高用无单位的比例**：正文 1.5，摘要 1.55，标题 1.4，引导行 1.2。侧栏简介用 2 倍行高，刻意排得疏松。
- **单行控件行高为 1**：导航项、筛选胶囊、分段开关，高度由内边距撑开。
- **字重三档**：400、500、600。500 给标题和选中项；600 只出现在列表视图的分类小标题上。没有 700。
- **汉字不设字距**；只有拉丁标题有极小的负字距，刊名 -0.03em。
- 等宽字体 Geist Mono 与 Adobe 的 Neue Haas Grotesk Display 已在主题里声明，但首页瀑布流没有用到。

## Layout & Spacing

```yaml
spacing:
  xs: 4px
  sm: 8px
  md: 12px
  lg: 16px
  xl: 20px
  2xl: 24px
  canvas-inset: 8px
  sidebar-width: 256px
  column-width: 272px
  panel-inset: 20px
  panel-bottom-inset: 108px
  list-max-width: 720px
  mobile-gutter: 14px
  mobile-bar-height: 56px
```

间距以 4px 为基本单位，高频值是 12px 与 16px。

- **画布与面板**：整页是一块 #F5F5F7 的画布，四周内缩 8px。左边是 256px 的侧栏（透明，直接坐在画布上），右边是一扇白色主面板，四角 16px，两者相隔 4px。主面板独立滚动，滚动条槽位常驻，滚动时不跳动。
- **侧栏**：内边距上 22、右 18、下 18、左 20，段间距 20px。自上而下三段：身份（刊名、简介、三个外链）、日期导航（最近 7 天加"更多"）、偏好（推广卡、虚线分隔、视图切换与主题按钮）。
- **面板内部**：内边距 20px，底部预留 108px 给悬浮筛选条。
- **瀑布流**：若干列并排，列宽 272px、列间距 12px，列内卡片间距 12px。每列是独立的纵向流，卡片高度可以各不相同——封面保留原始比例，不裁切。列数随面板宽度增减：1440 宽的视口 4 列，1740 宽 5 列，1024 宽 2 列。
- **卡片内部**：封面通栏 → 正文区内边距 16px → 引导行（分类 · 来源）→ 8px → 标题（一到两行）→ 摘要。
- **今日看点**：桌面端固定在最后一列的顶部，是一块浅灰底的摘要卡，下半部分是贴纸区。
- **列表视图**：内容收窄到 720px 居中。"今日看点"变成通栏的横幅卡，下间距 40px；每个分类一个 13px 的小标题，行是"左文字、右缩略图"的横向卡片，行间距 12px。
- **悬浮筛选条**：位于主面板内，距底 20px、左右各 20px，水平居中，高 48px。条下有一道 80px 高的渐变遮罩，把滚过的内容淡出。
- **视口不超过 1024px**：列表视图改为"720px 与可用宽度取小"。
- **视口不超过 649px**：画布变纯白、主面板不再有圆角、侧栏隐藏。顶部是固定的 56px 栏（左刊名 26px，右 36px 圆形抽屉按钮）。抽屉覆盖顶栏以下的整个屏幕，底色 #F5F5F7，依次是简介（14px）、外链、日期导航（行高 44px）、推广卡、视图切换与主题按钮。内容单列，间距 10px，左右边距 14px；筛选条贴底 16px、左右 12px；"今日看点"移到最前，贴纸区高 162px。

## Elevation & Depth

两层结构：浅灰画布，和浮在它上面的白色面板。面板本身不靠投影，靠与画布的色差说明"这是一层"。

- **卡片在白面板上用 1px #EBEBEB 描边定界，没有投影。** 悬停时描边加深到 #D9D9D9。
- **"今日看点"卡反过来**：没有描边，浅灰底 #F5F5F7。在白底上像一块垫了灰纸的区域，比周围的内容卡低半层。
- **投影只给真正浮在内容之上的元素**：
  - 悬浮筛选条：外投影 0 10px 24px 14% 黑，加顶部 1px 白色内高光和 20px 白色内发光，形成一点玻璃质感。
  - 贴纸标签：0 3px 10px 7% 黑，加 0 1px 3px 4% 黑。
  - 卡片上的"前往"圆钮：0 1px 2px 4% 黑。
  - 分段开关的选中块：0 1px 2px 6% 黑。
  - 列表的预览浮层：三层叠加的大投影，最远一层 0 20px 48px -10px 22% 黑。
- **深色主题把描边式的投影定义为 1px 白色光环**（8% 到 16% 不透明的白），因为在深底上黑投影看不见。

## Shapes

```yaml
rounded:
  xs: 5px
  sm: 10px
  md: 12px
  lg: 16px
  full: 9999px
```

- **没有直角**，连"今日看点"这样的灰底区块也是 12px 圆角。
- 内容卡片 12px；主面板与推广卡 16px；列表的预览浮层 10px；列表缩略图 5px（手机上 4px）。
- **所有控件都是全圆**：导航项、筛选胶囊、分段开关、圆形图标按钮、指示点、贴纸标签。
- 指示点高 5px：当前项是 16px 宽的短横条，其余是小圆点。

## Motion & Micro-interaction

```yaml
motion:
  state-change:
    duration: 150ms
    delay: 0ms
    timingFunction: cubic-bezier(0.22, 1, 0.36, 1)
  card-enter:
    duration: 400ms
    delay: 0ms
    timingFunction: cubic-bezier(0.22, 1, 0.36, 1)
  image-zoom:
    duration: 500ms
    delay: 0ms
    timingFunction: cubic-bezier(0.22, 1, 0.36, 1)
  go-reveal:
    duration: 250ms
    delay: 0ms
    timingFunction: cubic-bezier(0.22, 1, 0.36, 1)
  dot-grow:
    duration: 300ms
    delay: 0ms
    timingFunction: cubic-bezier(0.22, 1, 0.36, 1)
  promo-slide:
    duration: 450ms
    delay: 0ms
    timingFunction: cubic-bezier(0.33, 0.55, 0.2, 1)
  popover:
    duration: 240ms
    delay: 0ms
    timingFunction: cubic-bezier(0.2, 0.9, 0.3, 1.15)
```

一条主曲线管几乎全部过渡：cubic-bezier(0.22, 1, 0.36, 1)，强减速，默认时长 150ms。颜色、底色、描边的细微变化都是这个节奏。

- **入场**：卡片用 400ms 淡入并上浮 8px。同一列里按位置依次延迟 80ms、120ms、160ms，形成"轻轻落下"的错落，不是整齐的一起出现。
- **悬停卡片**：封面在 500ms 内放大到 1.03；右下角浮出"前往"圆钮——从 25% 大小、4px 模糊、全透明，长到原大并清晰（250ms）；描边同时加深。
- **骨架屏**：封面未加载时显示 1.15s 循环的微光扫过；加载完成后图片 250ms 淡入。
- **贴纸标签**：由物理模拟驱动，可以抓取、拖动，松手后落下堆叠；散落姿态彼此重叠、倾角各异。这是全站唯一"好玩"的动效，不要复制到别处。
- **轮播**：推广卡 450ms 横向滑动（cubic-bezier(0.33, 0.55, 0.2, 1)），指示点 300ms 变宽。
- **列表预览浮层**：240ms 淡入，带一点回弹（cubic-bezier(0.2, 0.9, 0.3, 1.15)）和轻微旋转；按下行时整行缩到 0.995。
- **触屏**：所有悬停效果包在"支持悬停"的媒体条件里，触屏上不显示"前往"圆钮和预览浮层。
- **prefers-reduced-motion**：卡片没有入场动画，封面不缩放，轮播停用、只显示当前卡。

## Voice & Content Strategy

- **语言**：全中文。产品名与专有名词保留英文原文（GLM-5.4、Linear、Ladybird）。
- **语气**：编辑式的，口语但不轻佻。作者以第一人称出现——"我是伊恩，产品设计师，正在尝试做各种「杂七杂八」的产品"。每条收录都附一句编辑判断，而不是照抄原标题："官方称不增参数逼近 2-3T 效果，待三方复测。"
- **标点**：引用用直角引号「」；句末用全角句号；并列项之间用中点分隔，两侧各空一格。
- **标题**：一句话说清"谁做了什么"，数字与型号直接写进去，不设固定字数，一至两行。例："MiniMax 发 H3 系列并披露百日数据"。
- **引导行的固定格式**："分类 · 来源域名"。来源写域名而不是站点名（z.ai、techcrunch.com）。
- **日期**：写作"11月24日"——月和日用汉字，数字用阿拉伯数字，中间不留空格；右侧以 12px 灰字标注当日条数。浏览器标题的格式是"2026-11-24 · 伊恩日刊"。
- **分类名是 2 到 4 字的名词短语**：全部、AI 资讯、一人公司、产品设计、审美提升、产品营销、AI 协作。
- **行动按钮**：动词短语加右上箭头——了解早鸟课程、去圈子看看、聊聊合作；次级入口是"查看更多""更多"。视图切换写作"瀑布流 / 列表"。

## Components

```yaml
components:
  main-panel:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
  card:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.card-title}"
    rounded: "{rounded.md}"
  card-rule:
    backgroundColor: "{colors.line}"
    height: 1px
  card-rule-hover:
    backgroundColor: "{colors.line-hover}"
  card-media:
    backgroundColor: "{colors.neutral}"
  card-kicker:
    textColor: "{colors.text-subtle}"
    typography: "{typography.kicker}"
  card-summary:
    textColor: "{colors.text-muted}"
    typography: "{typography.body-sm}"
  go-button:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    rounded: "{rounded.full}"
    size: 32px
  summary-card:
    backgroundColor: "{colors.neutral}"
    textColor: "{colors.primary}"
    rounded: "{rounded.md}"
    padding: 16px 16px 0px
  summary-title:
    textColor: "{colors.primary}"
    typography: "{typography.section-title}"
  summary-date:
    textColor: "{colors.text-subtle}"
    typography: "{typography.body-sm}"
  sticker-pill:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text-muted}"
    typography: "{typography.sticker}"
    rounded: "{rounded.full}"
    padding: 5px 12px
  wordmark:
    textColor: "{colors.primary}"
    typography: "{typography.wordmark}"
  bio:
    textColor: "{colors.primary}"
    typography: "{typography.bio}"
  sidebar-link:
    textColor: "{colors.text-muted}"
    typography: "{typography.body-sm}"
  nav-item:
    textColor: "{colors.text-muted}"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: 36px
    padding: 8px 14px
  nav-item-hover:
    backgroundColor: "{colors.hover-tint}"
  nav-item-active:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.label-md-medium}"
  nav-count:
    textColor: "{colors.text-subtle}"
    typography: "{typography.label-sm}"
  promo-panel:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
    padding: 16px 16px 0px
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.label-sm-medium}"
    rounded: "{rounded.full}"
    height: 32px
  promo-dot:
    backgroundColor: "{colors.primary}"
    height: 5px
    width: 16px
  promo-dot-inactive:
    backgroundColor: "{colors.line-strong}"
    size: 5px
  divider:
    backgroundColor: "{colors.line-strong}"
    height: 1px
  segmented-control:
    backgroundColor: "{colors.neutral}"
    rounded: "{rounded.full}"
    height: 32px
    padding: 2.5px
  segment:
    textColor: "{colors.text-subtle}"
    typography: "{typography.label-sm-medium}"
    rounded: "{rounded.full}"
    height: 25px
    padding: 0px 6px
  segment-active:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
  icon-button:
    backgroundColor: "{colors.neutral}"
    rounded: "{rounded.full}"
    size: 32px
  pill-bar:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.full}"
    height: 48px
    padding: 4px
  pill-bar-rule:
    backgroundColor: "{colors.line}"
    height: 1px
  pill:
    textColor: "{colors.text-muted}"
    typography: "{typography.label-md-medium}"
    rounded: "{rounded.full}"
    height: 38px
    padding: 0px 14px
  pill-hover:
    backgroundColor: "{colors.hover-tint}"
  pill-active:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
  list-row:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.card-title}"
    rounded: "{rounded.md}"
    padding: 16px
  list-thumb:
    backgroundColor: "{colors.neutral}"
    rounded: "{rounded.xs}"
    height: 44px
    width: 78px
  list-heading:
    textColor: "{colors.primary}"
    typography: "{typography.list-heading}"
  popover:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.sm}"
    padding: 2px
    width: 248px
    height: 140px
  text-error:
    textColor: "{colors.error}"
  main-panel-dark:
    backgroundColor: "{colors.surface-dark}"
    rounded: "{rounded.lg}"
  card-dark:
    backgroundColor: "{colors.surface-dark}"
    textColor: "{colors.primary-dark}"
    typography: "{typography.card-title}"
    rounded: "{rounded.md}"
  card-rule-dark:
    backgroundColor: "{colors.line-dark}"
    height: 1px
  card-rule-hover-dark:
    backgroundColor: "{colors.line-hover-dark}"
  card-media-dark:
    backgroundColor: "{colors.neutral-dark}"
  card-kicker-dark:
    textColor: "{colors.text-subtle-dark}"
    typography: "{typography.kicker}"
  card-summary-dark:
    textColor: "{colors.text-muted-dark}"
    typography: "{typography.body-sm}"
  go-button-dark:
    backgroundColor: "{colors.surface-raised-dark}"
    textColor: "{colors.primary-dark}"
    rounded: "{rounded.full}"
    size: 32px
  summary-card-dark:
    backgroundColor: "{colors.neutral-dark}"
    textColor: "{colors.primary-dark}"
    rounded: "{rounded.md}"
  sticker-pill-dark:
    backgroundColor: "{colors.sticker-dark}"
    textColor: "{colors.sticker-ink-dark}"
    typography: "{typography.sticker}"
  nav-item-hover-dark:
    backgroundColor: "{colors.hover-tint-dark}"
  nav-item-active-dark:
    backgroundColor: "{colors.surface-selected-dark}"
    textColor: "{colors.primary-dark}"
  divider-dark:
    backgroundColor: "{colors.line-strong-dark}"
    height: 1px
  pill-bar-dark:
    backgroundColor: "{colors.bar-dark}"
    rounded: "{rounded.full}"
  pill-bar-rule-dark:
    backgroundColor: "{colors.bar-line-dark}"
    height: 1px
  pill-active-dark:
    backgroundColor: "{colors.primary-dark}"
    textColor: "{colors.on-primary-dark}"
  button-primary-dark:
    backgroundColor: "{colors.primary-dark}"
    textColor: "{colors.on-primary-dark}"
```

描边与分隔线的颜色以 1px 高的 `*-rule`、`divider` 组件表达，因为规范的组件属性里没有边框一项。

- **内容卡片**：白底、1px #EBEBEB 描边、12px 圆角、无投影。封面通栏并保留原始比例；正文区内边距 16px，依次是 11px 的引导行（分类、3px 小圆点、来源域名）、14px 的标题、13px 的摘要。悬停时描边加深、封面放大、右下角浮出"前往"圆钮（32px，白底，1px 描边，右上箭头）。
- **今日看点卡**：浅灰底、无描边、12px 圆角。上半是 15px 的栏目标题（右侧对齐日期）和一段 13px 的导语；下半 165px 高的贴纸区。
- **贴纸标签**：白色胶囊、muted 灰文字、小投影，鼠标是抓取手势；抓起时文字变墨色、投影加深。
- **日期导航项**：全圆胶囊，高 36px，左右内边距 14px。选中项白底、墨色、500 字重；未选中为 muted 灰；悬停浮出 6% 黑底并变墨色。右侧的条数是 12px 的 subtle 灰。
- **筛选胶囊条**：白色胶囊容器，高 48px、内边距 4px、1px 描边、玻璃质感的投影。里面的胶囊高 38px、13px 文字、500 字重；选中的胶囊墨底白字，其余 muted 灰。
- **推广卡**：白底、16px 圆角。标题 14px / 500，两行 13px 说明，通栏的墨底白字按钮（高 32px、12px 文字），下方三个指示点。
- **分段开关**：浅灰底的胶囊容器（高 32px、1px 描边、内边距 2.5px），选中块是白底小胶囊加 1px 投影，文字 12px / 500；未选中项文字 subtle 灰。
- **图标按钮**：32px 正圆，浅灰底、1px 描边、subtle 灰图标；悬停时图标变墨色、描边加深。
- **列表行**：白底卡片，内边距 16px，左侧是标题与一句判断，右侧是 78×44 的缩略图（5px 圆角、1px 描边）。悬停时右侧弹出 248×140 的预览浮层。
- **图标**：线性图标（lucide），约 14–16px，细描边，颜色跟随文字。
- **焦点**：2px 墨色描边；胶囊偏移 2px，控件偏移 1px，推广卡用内缩 3px 的描边。

## Do's and Don'ts

- Do 让封面图提供色彩，界面保持墨、灰、白。
- Do 用半透明黑色调做悬停底色，用白与墨的反相表示选中。
- Do 保持瀑布流的列各自独立、图片保留原始比例——卡片高度不齐是特征，不是缺陷。
- Do 把投影留给真正浮起的东西：筛选条、贴纸、预览浮层。
- Don't 给内容卡片加投影，它们只靠 1px 描边定界。
- Don't 用 #86868B 写需要通读的文字。
- Don't 设置 15px 以上的内容标题，或引入 700 字重。
- Don't 把贴纸标签的物理动效复制到别处，也不要在触屏上显示"前往"圆钮。
- Don't 把手写体用在刊名以外的地方。
