const DURATION_MS = 650

// Where the reveal starts: the control that changed the theme, or the middle of the screen.
function origin(): { x: number; y: number } {
  const el = document.activeElement
  if (el instanceof HTMLElement && el !== document.body) {
    const rect = el.getBoundingClientRect()
    return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
  }
  return { x: window.innerWidth / 2, y: window.innerHeight / 2 }
}

/*
 * Runs `change` inside a view transition and grows the new theme out of a
 * circle, instead of every colour flipping at once. Element colour
 * transitions are paused meanwhile (the theme-switching class), or the
 * "after" snapshot would be taken halfway through them and the reveal would
 * end on a jump. Without View Transitions support, or under reduced motion,
 * the change just applies.
 */
export function revealTheme(change: () => void) {
  const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (!('startViewTransition' in document) || reduce) {
    change()
    return
  }

  const root = document.documentElement
  const { x, y } = origin()
  const radius = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y))

  root.style.setProperty('--reveal-x', `${x}px`)
  root.style.setProperty('--reveal-y', `${y}px`)
  root.classList.add('theme-switching')
  const transition = document.startViewTransition(async () => {
    change()
    // Let the charts rebuild before the snapshot. Not requestAnimationFrame: no frame runs until this resolves.
    await new Promise((resolve) => setTimeout(resolve, 30))
  })
  transition.ready
    .then(() => {
      root.animate(
        { clipPath: [`circle(0px at ${x}px ${y}px)`, `circle(${radius}px at ${x}px ${y}px)`] },
        { duration: DURATION_MS, easing: 'cubic-bezier(0.4, 0, 0.2, 1)', fill: 'forwards', pseudoElement: '::view-transition-new(root)' },
      )
    })
    .catch(() => {})
  void transition.finished.finally(() => {
    root.classList.remove('theme-switching')
    root.style.removeProperty('--reveal-x')
    root.style.removeProperty('--reveal-y')
  })
}
