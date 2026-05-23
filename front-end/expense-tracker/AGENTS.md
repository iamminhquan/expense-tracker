<!-- BEGIN:nextjs-agent-rules -->
# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` before writing any code. Heed deprecation notices.
<!-- END:nextjs-agent-rules -->

# Frontend UI Rules

## Product Direction

Build the actual budget and expense tracking experience, not a marketing page. The first screen should help users sign in, review their account, scan financial data, or complete a money-management task. Avoid oversized hero sections, decorative filler, and generic SaaS copy unless the route is explicitly a public landing page.

The interface should feel quiet, practical, and trustworthy. Prioritize clear hierarchy, readable numbers, predictable controls, and fast repeated use over visual novelty.

## Layout & Composition

Use full-width page sections with constrained inner content. Do not nest cards inside cards, and do not turn every section into a floating card. Reserve cards for individual repeated items, summaries, forms, modals, and genuinely framed tools.

Keep border radii restrained. Use `rounded-md` by default for panels, inputs, and cards unless an existing component requires a different radius.

Use stable responsive dimensions for fixed-format UI such as auth panels, dashboards, charts, tables, toolbar buttons, and summary tiles. Text, hover states, loading labels, and dynamic values must not cause layout shift.

On mobile, stack primary content before secondary controls. Form inputs and buttons must remain easy to tap and must not overflow their container.

## Components & Styling

Prefer existing components in `components/ui/` and helpers in `lib/` before creating new primitives. Keep route-specific UI in `app/`; move reusable UI to `components/`.

Use Tailwind utility classes consistently with the app tokens from `app/globals.css`: `background`, `foreground`, `card`, `muted`, `border`, `primary`, `destructive`, and related foreground colors. Avoid hardcoded one-off colors unless they communicate domain meaning such as success, warning, expense, or income.

Use `cn()` for conditional class names. Keep component props typed and explicit.

Use lucide icons for recognizable actions such as login, logout, save, edit, delete, filter, search, calendar, wallet, chart, and settings. Icon-only buttons need an accessible label or tooltip.

Do not add decorative gradient orbs, bokeh blobs, or abstract SVG filler. If a visual asset is needed, it should clarify the product, data, or task.

## Forms & Interaction

Every form must cover normal, loading, success, and error states. Disable controls while requests are in flight when duplicate submissions would be harmful.

Use semantic labels for inputs. Set appropriate `type`, `autoComplete`, and validation hints. Error messages should be specific enough to fix the input without exposing sensitive backend details.

Keep authentication and money-related flows conservative: do not hide failed network/API states, do not silently drop user input, and do not show a signed-in state until the API response confirms it.

For destructive or irreversible actions, require a clear confirmation path and show the affected item name or amount.

## Accessibility & Readability

Text must fit within its container at common mobile and desktop widths. Do not use viewport-width-based font sizing. Keep letter spacing normal unless matching an existing style.

Maintain visible focus states and keyboard access for buttons, inputs, tabs, segmented controls, and menus.

Use sufficient contrast for muted text, borders, charts, and status messages. Do not communicate important state by color alone.

Use concise labels that match the user's task. Avoid visible text explaining implementation details, component behavior, keyboard shortcuts, or design decisions.

## Data Display

Financial values should be aligned and easy to compare. Use tabular layouts or consistent grid columns for lists of transactions, budgets, balances, and category totals.

Format dates, currency, and percentages consistently. Keep raw API field names out of the UI.

Empty states should provide the next useful action, such as adding an expense, creating a budget, or signing in.

## Verification

After UI changes, run:

- `bun run lint`
- `bun run build`

For visible interaction changes, start the app with `bun dev` and inspect the affected route at `http://localhost:3000`. Check at least one mobile-width and one desktop-width layout when the change affects responsive structure.
