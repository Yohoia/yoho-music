---
version: alpha
name: Topinspo
description: "设计灵感站 Topinspo 的视觉规范：纯白展墙、衬线体作品名、无衬线元信息、一处橄榄绿。提取自 www.topinspo.design（2026-11-25）。"
---

<!-- 提取自线上站点。令牌是站点样式表里的作者声明值；尺寸、对比度与断点行为来自浏览器实测。 -->

## Overview

把 Topinspo 想成美术馆的一整面展墙：作品挂在纯白的墙上，每件作品下面贴一枚博物馆式标签——衬线体的作品名，再一行无衬线的"作者 · 类别"。界面自己几乎不出声；颜色、动感与视觉冲击全部来自被陈列的作品。

- **是什么**：设计师 Ian 维护的设计灵感库，按 Web、Branding、Product、Motion 等分类浏览 UI、品牌、三维与印刷作品。同一框架下还有设计工具、设计 Skills、设计词汇等栏目。
- **给谁看**：来找参考的设计师与设计工程师。他们知道自己要什么，页面的职责是让作品尽快被看到，而不是介绍自己。
- **应有的感受**：安静、克制、有编辑判断。像一本印得很好的作品目录，而不是社交产品的信息流。
- **尺度感**：界面文字几乎只有 14px 一种字号，页面主标题也只到 20px。层级靠"衬线 / 无衬线"、"墨色 / 灰色"、"有无底色"来表达，不靠放大。
- **结构感**：这是应用式界面而不是文档——左栏导航固定，右侧内容区独立滚动，整页不滚动。

## Colors

```yaml
colors:
  primary: "#292929"
  on-primary: "#FCFCF8"
  secondary: "#5B6F00"
  tertiary: "#788C15"
  neutral: "#F4F4F1"
  surface: "#FFFFFF"
  ink-strong: "#1E1E1E"
  ink-soft: "#4E4D4B"
  muted: "#72726E"
  faint: "#ACADA8"
  line: "#0000001A"
  line-strong: "#D5D5D2"
  hover-tint: "#1516180A"
  selected-tint: "#1516180F"
  accent-hover: "#5B6F00E6"
```

整套颜色是一组略偏暖的中性灰加一个橄榄绿，没有大面积品牌色。橄榄绿只出现在三处：logo 方块、选中分类下方的 2px 短线、键盘焦点环。站点没有深色模式。

- **Primary（#292929）**：墨色。默认文字、导航文字、作者名，也是主按钮的底色，配奶白文字 #FCFCF8——不是纯白，与暖灰底色相呼应。
- **Ink strong（#1E1E1E）与 Ink soft（#4E4D4B）**：墨色的两档变体。深的一档只给衬线体标题；浅的一档是未选中的分类标签，也是卡片标题悬停时的颜色——悬停时标题反而变浅，是刻意的一步退让。
- **Muted（#72726E）**：类别、未选中的导航项、侧栏简介。对白底 4.8:1，是可读文字的下限。
- **Faint（#ACADA8）**："10k+ followers" 一类附注和禁用项。对白底只有 2.3:1，只承载丢了也不影响理解的信息。
- **Neutral（#F4F4F1）**：作品媒体的底色，比白略暖的灰。
- **Surface（#FFFFFF）**：页面底色，纯白。
- **Secondary（#5B6F00）与 Tertiary（#788C15）**：橄榄绿和更亮一档的黄绿。前者用于强调型主按钮和焦点环，后者用于选中短线和 logo 方块。
- **Line（#000000，10%）与 Line strong（#D5D5D2）**：发丝线。浅的一档做分类栏底线和媒体内侧描边，深的一档做胶囊按钮的 0.5px 描边、侧栏链接下划线和滚动条。
- **Hover tint 与 Selected tint**：悬停和选中的底色，分别是 #151618 的 4% 与 6%。它们是半透明的，叠在任何底色上都成立；在白底上等效于 #F6F6F6 与 #F1F1F1。

## Typography

```yaml
typography:
  page-title:
    fontFamily: "'Roboto Serif', Georgia, serif"
    fontSize: 20px
    fontWeight: 400
    lineHeight: 26px
    letterSpacing: -0.01em
  page-title-zh:
    fontFamily: "'Hanken Grotesk', 'PingFang SC', system-ui, sans-serif"
    fontSize: 20px
    fontWeight: 500
    lineHeight: 26px
    letterSpacing: -0.01em
  card-title:
    fontFamily: "'Roboto Serif', Georgia, serif"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 22px
    letterSpacing: -0.01em
  section-title-zh:
    fontFamily: "'Hanken Grotesk', 'PingFang SC', system-ui, sans-serif"
    fontSize: 14px
    fontWeight: 500
    lineHeight: 22px
  body-md:
    fontFamily: "'Hanken Grotesk', 'PingFang SC', system-ui, sans-serif"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 24px
    letterSpacing: 0.01em
  label-md:
    fontFamily: "'Hanken Grotesk', 'PingFang SC', system-ui, sans-serif"
    fontSize: 14px
    fontWeight: 400
    lineHeight: 18px
    letterSpacing: 0.01em
  label-md-medium:
    fontFamily: "'Hanken Grotesk', 'PingFang SC', system-ui, sans-serif"
    fontSize: 14px
    fontWeight: 500
    lineHeight: 18px
    letterSpacing: 0.01em
  label-sm:
    fontFamily: "'Hanken Grotesk', 'PingFang SC', system-ui, sans-serif"
    fontSize: 12px
    fontWeight: 400
    lineHeight: 16px
    letterSpacing: 0.01em
```

两款自托管的可变字体各管一件事，这是本规范最关键的一条：

- **Roboto Serif 只用于"作品名"**：页面主标题、卡片标题、推广卡标题。标题不靠字号大，靠换字体——页面主标题 20px，卡片标题 14px，与正文、导航、按钮同号。
- **Hanken Grotesk 管其余全部界面文字**：导航、分类、按钮、元信息、简介。
- **字重只有两档**：400 与 500。500 表示"当前选中"或"可点击的强调"——当前导航项、分类标签、胶囊按钮上的文字。没有加粗，没有斜体，没有全大写。
- **行高写死为像素值**：单行界面文字 18px，说明段落 24px，衬线标题 22px（主标题 26px）。
- **字距极小**：无衬线 +0.01em，衬线 -0.01em。不要自行加大。
- **中文界面**：标题改用 Hanken Grotesk 字体栈、500 字重（实测中文落回系统字体 PingFang SC），衬线只留给仍是英文的作品名；导航项之间留 4px，比英文的 2px 宽。
- 字体文件随站点自托管（hanken-grotesk-variable.woff2、roboto-serif-variable.woff2），两款都是可变字体。

## Layout & Spacing

```yaml
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
  2xl: 48px
  shell-max-width: 1200px
  sidebar-width: 220px
  content-max-width: 964px
  content-gutter: 48px
  content-inset-top: 40px
  content-inset-bottom: 96px
  grid-gap-x: 32px
  grid-gap-y: 48px
  mobile-bar-height: 56px
  mobile-gutter: 20px
```

间距以 4px 为基本单位。

- **应用式框架**：外壳高度锁定为视口高度，内容整体居中且最宽 1200px。左栏 220px 固定，右侧主区占满余下宽度并独立滚动，页面本身不滚动。
- **侧栏**：内边距上 40px、下 28px，栏内三段（身份、导航、推广位）之间隔 32px。导航项之间只隔 2px，靠胶囊底色而不是间距来区分。
- **内容壳**：最大宽度 964px，靠左贴着主区，内边距上 40、右 24、下 96、左 48。左右刻意不对称——左侧多出的 24px 让内容与侧栏之间留一条呼吸带。
- **头部**：一行 36px 高，左标题右操作区，操作区内两个胶囊相隔 8px。分类栏距头部 20px，高 40px，项间距 24px，底部一条 1px 线。
- **作品网格**：3 列等宽，列间距 32px、行间距 48px。行距明显大于列距，让每一横排读起来像画册里的一行图版。
- **卡片纵向节奏**：媒体（4:3）→ 16px → 标题（单行，22px）→ 8px → 元信息行（20px）。
- **断点依据容器而不是视口**：内容壳是一个具名容器，宽度不超过 720px 时网格变 2 列，不超过 460px 时变 1 列，不超过 560px 时隐藏头部右侧的描边胶囊。
- **视口不超过 900px**：侧栏收起，换成固定在顶部的 56px 栏（白底 92% 不透明、12px 背景模糊、底部 0.5px 线）；内容内边距改为上 24、左右 20、下 48px（加安全区）；分类栏可横向滚动且不显示滚动条；导航进入全屏抽屉，抽屉内导航项高 44px。
- **桌面窗口较矮（高度不超过 800px）**：侧栏自动收紧，栏内间距 16px、导航项最小高度 32px，保证推广位不被挤出屏幕。

## Elevation & Depth

主信息流是平的：没有投影，也没有带边框的"卡片盒子"。一张卡片就是"一张图加一枚标签"，直接放在白底上。层次靠三样东西：

- **底色温差**：作品媒体坐在暖灰 #F4F4F1 上，与纯白页面拉开一层。
- **发丝线**：媒体内侧一圈 0.5px、10% 黑的内描边，让浅色作品在浅灰底上仍有轮廓；胶囊按钮用 0.5px #D5D5D2 描边。
- **半透明底色**：悬停 4%、选中 6% 的黑，把当前项轻轻托起来。

投影令牌只有两档，都预留给浮在内容之上的浮层，内容流里不出现：菜单类浮层用 `0 1px 2px 4% 黑，0 6px 20px 6% 黑`；更大的弹出面板用 `0 20px 25px -5px 10% 黑，0 8px 10px -6px 10% 黑`，圆角 16px。移动端顶栏靠 12px 背景模糊浮在内容之上。

## Shapes

```yaml
rounded:
  none: 0px
  xs: 4px
  sm: 8px
  lg: 16px
  full: 9999px
```

- 圆角只有四个值，分工明确。**0**：所有文字和网格单元。**4px**：20px 的作者头像。**8px**：作品媒体、推广卡——一切"内容容器"。**16px**：弹出面板。
- **所有可点击的控件都是全圆胶囊**：导航项、头部按钮、主按钮、指示点。**所有内容容器都是 8px**。两种圆角不要混用。
- 作品媒体固定 4:3，图片与视频一律裁满（object-fit: cover）。
- 推广卡的指示点高 5px：当前项是 16px 宽的短横条，其余是小圆点。

## Motion & Micro-interaction

```yaml
motion:
  state-change:
    duration: 150ms
    delay: 0ms
    timingFunction: cubic-bezier(0.4, 0, 0.2, 1)
  indicator-change:
    duration: 200ms
    delay: 0ms
    timingFunction: cubic-bezier(0.4, 0, 0.2, 1)
  press:
    duration: 75ms
    delay: 0ms
    timingFunction: cubic-bezier(0.4, 0, 0.2, 1)
  dot-grow:
    duration: 300ms
    delay: 0ms
    timingFunction: cubic-bezier(0.4, 0, 0.2, 1)
  promo-slide:
    duration: 450ms
    delay: 0ms
    timingFunction: cubic-bezier(0.33, 0.55, 0.2, 1)
```

动效只做一件事：提示"这里可以点"。全部是颜色、底色、线色的淡入淡出，150ms（快）或 200ms（常规），统一使用 cubic-bezier(0.4, 0, 0.2, 1)。

- **会动的**：导航项的底色与文字色；分类标签的文字色与下划线色；胶囊按钮的底色；卡片标题的墨色（悬停变为 #4E4D4B）；侧栏链接下划线的颜色；logo 的不透明度（悬停降到 70%）。
- **不动的**：卡片不抬升、不放大、不加投影；媒体不缩放；没有入场动画，没有滚动视差，没有循环播放的装饰动效。
- **仅有的两处"结构"动效**都在侧栏推广位：卡片 450ms 横向滑动（cubic-bezier(0.33, 0.55, 0.2, 1)），指示点 300ms 变宽。
- **最快的回应只有 75ms**：强调型主按钮的悬停底色变化。推广卡被悬停时，卡内的主按钮降到 90% 不透明，提示整张卡都可以点。
- 尊重 prefers-reduced-motion：过渡与动画缩短到近乎 0，推广轮播停止滑动、只显示当前卡。

## Voice & Content Strategy

- **语言**：界面默认英文，右上角一键切换中文。切换只翻译界面与分类名，作品标题保留原文。
- **语气**：简短、名词化、不感叹。像标签，不像句子。
- **导航命名**是固定句式"设计 + 对象"：Design Inspiration、Design Tools、Design Skills、Design Vocabulary、Design Components、Design Prompts；中文为设计灵感、设计工具、设计 Skills、设计词汇、设计组件、设计提示词——没有通行译名时保留英文词。
- **大小写**：英文标题与按钮用句首大写（Design Inspiration、Enroll now）。中文不加句号。
- **分类词表固定八个**：All、Web、Branding、Product、Motion、Illustration、3D、Print，对应全部、网页、品牌、产品、动效、插画、3D、平面。卡片的类别只能取自这张表。
- **卡片标签的固定格式**：第一行作品名（衬线），第二行"@作者 · 类别"（无衬线）。分隔符是中点，两侧各空一格。
- **数字**：英文写"10k+ followers""10k+ stars"，中文写"1万+ 粉丝""1万+ Star"。
- **行动按钮**：动词开头，以右上箭头图标结尾——Enroll now、Join us；中文"立即加入""加入圈子"。次级入口用 More（更多）。
- **未上线的栏目**：文字变灰、不可点击，悬停浮出"Coming soon"提示。
- **空状态**：一段 muted 灰的说明文字，没有插图。

## Components

```yaml
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.label-md-medium}"
    rounded: "{rounded.full}"
    height: 32px
    padding: 0px 14px
  button-accent:
    backgroundColor: "{colors.secondary}"
    textColor: "{colors.on-primary}"
    typography: "{typography.label-md-medium}"
    rounded: "{rounded.full}"
    height: 40px
    padding: 0px 18px
  button-accent-hover:
    backgroundColor: "{colors.accent-hover}"
  button-outline:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    typography: "{typography.label-md-medium}"
    rounded: "{rounded.full}"
    height: 36px
    padding: 0px 16px
  button-outline-hover:
    backgroundColor: "{colors.hover-tint}"
  button-icon:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    rounded: "{rounded.full}"
    size: 36px
  nav-item:
    textColor: "{colors.muted}"
    typography: "{typography.label-md}"
    rounded: "{rounded.full}"
    height: 38px
    padding: 10px 14px
  nav-item-hover:
    backgroundColor: "{colors.hover-tint}"
  nav-item-active:
    backgroundColor: "{colors.selected-tint}"
    typography: "{typography.label-md-medium}"
  nav-item-disabled:
    textColor: "{colors.faint}"
  tab:
    textColor: "{colors.ink-soft}"
    typography: "{typography.label-md-medium}"
    height: 40px
    padding: 10px 0px
  tab-hover:
    textColor: "{colors.primary}"
  tab-active:
    textColor: "{colors.primary}"
  tab-rule:
    backgroundColor: "{colors.line}"
    height: 1px
  tab-indicator:
    backgroundColor: "{colors.tertiary}"
    height: 2px
  tab-indicator-hover:
    backgroundColor: "{colors.line-strong}"
  card-media:
    backgroundColor: "{colors.neutral}"
    rounded: "{rounded.sm}"
  card-title:
    textColor: "{colors.ink-strong}"
    typography: "{typography.card-title}"
  card-title-hover:
    textColor: "{colors.ink-soft}"
  card-meta:
    textColor: "{colors.muted}"
    typography: "{typography.label-md}"
  avatar:
    rounded: "{rounded.xs}"
    size: 20px
  sidebar-bio:
    textColor: "{colors.ink-soft}"
    typography: "{typography.body-md}"
  sidebar-stat:
    textColor: "{colors.faint}"
    typography: "{typography.body-md}"
  promo-panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.muted}"
    rounded: "{rounded.sm}"
    padding: 20px 20px 0px
  promo-dot:
    backgroundColor: "{colors.primary}"
    height: 5px
    width: 16px
```

描边与分隔线的颜色以 1px 或 2px 高的 `*-rule`、`tab-indicator` 组件表达，因为规范的组件属性里没有边框一项。

- **作品卡**：一个整块可点的区域，无外框、无底色、无投影。媒体 4:3、8px 圆角、内侧 0.5px 发丝描边；下方 16px 是单行衬线标题（超出省略），再 8px 是元信息行：20px 的头像（4px 圆角）、@作者名（墨色）、中点、类别（muted）。悬停时只有标题变色。
- **分类标签**：纯文字按钮。选中项墨色加底部 2px 黄绿短线；未选中为 #4E4D4B；悬停时文字变墨色，短线出现为 #D5D5D2。
- **导航项**：全圆胶囊，高 38px，左右内边距 14px。选中项 6% 黑底、500 字重、墨色文字；悬停 4% 黑底、墨色文字；即将上线的项文字 #ACADA8、不可点击。
- **胶囊按钮**：主按钮墨底奶白、高 32px；描边按钮白底、0.5px #D5D5D2 描边、高 36px、右侧带右上箭头；图标按钮是 36px 正圆、13px 文字（语言切换）。悬停时描边按钮与图标按钮浮出 4% 黑底；橄榄绿的强调型主按钮悬停时降为 90% 不透明。
- **侧栏链接行**：高 24px。名称带下划线（#D5D5D2，偏移 3px），右侧跟一段 #ACADA8 的统计；悬停时下划线变成文字色。
- **推广卡**：220px 宽的白底面板，0.5px 10% 黑描边、8px 圆角。内容依次是衬线标题、两到三行说明、通栏主按钮；下方居中三个指示点。多张卡横向轮播。
- **图标**：线性，约 16px，细描边，颜色跟随文字；外链一律用右上箭头。logo 是 18px 的橄榄绿圆角方块，里面一个奶白色的 T。
- **焦点**：所有可聚焦元素显示 2px 橄榄绿（#5B6F00）外描边，偏移 2px。
- **滚动条**：内容区用细滚动条，滑块 #D5D5D2；侧栏不显示滚动条。

## Do's and Don'ts

- Do 让作品图提供全部色彩，界面只保留白、灰、墨三色。
- Do 用换字体（衬线 / 无衬线）和灰度来表达层级，字号留在 14px。
- Do 保持行距大于列距（48 / 32），维持画册式的成排节奏。
- Do 把橄榄绿留给焦点环、选中短线和 logo。
- Don't 给作品卡加边框、底色或投影，也不要在悬停时抬升或放大卡片。
- Don't 使用 400 和 500 以外的字重，不要全大写，不要拉大字距。
- Don't 用 #ACADA8 承载必须读到的信息（对白底仅 2.3:1）。
- Don't 在界面里加入大面积品牌色、渐变或装饰图形。
- Don't 把卡片元信息排成等宽字体——它是 Hanken Grotesk，等宽字体只用于代码片段。
