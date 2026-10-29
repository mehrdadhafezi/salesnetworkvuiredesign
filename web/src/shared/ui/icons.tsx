// Minimal stroke icons for status and state (SN-207: status = label + icon + tone).
const paths = {
  check: 'M5 12l4 4 10-10',
  alert: 'M12 8v5M12 16.5v.5M10.3 3.8L2.6 17.5A2 2 0 004.3 20.5h15.4a2 2 0 001.7-3l-7.7-13.7a2 2 0 00-3.4 0z',
  x: 'M6 6l12 12M18 6L6 18',
  question: 'M9.5 9a2.5 2.5 0 115 .5c0 1.5-2.5 2-2.5 3.5M12 17v.5M12 3a9 9 0 100 18 9 9 0 000-18z',
  split: 'M4 12h6l4-6h6M14 18h6M10 12l4 6',
  lock: 'M7 11V8a5 5 0 0110 0v3M6 11h12v9H6z',
  ban: 'M5.6 5.6l12.8 12.8M12 3a9 9 0 100 18 9 9 0 000-18z',
  inbox: 'M3 13h5l2 3h4l2-3h5M5 5h14l2 8v6H3v-6z',
  menu: 'M4 7h16M4 12h16M4 17h16',
} as const

export type IconName = keyof typeof paths

export function Icon({ name }: { name: IconName }) {
  return (
    <svg className="i" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
      <path d={paths[name]} />
    </svg>
  )
}
