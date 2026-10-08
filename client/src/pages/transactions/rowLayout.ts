/* Every desktop row uses the same columns, so the list lines up without a header row.
   The edit row's date column stays at 168px or more: Firefox clips the year below that. */
export const DESKTOP_ROW_GRID = 'grid grid-cols-[116px_170px_minmax(0,1fr)_150px_172px] items-center gap-4'
export const DESKTOP_EDIT_GRID = 'grid grid-cols-[168px_190px_minmax(0,1fr)_160px_auto] items-center gap-3'
