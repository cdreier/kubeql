const POD_COLORS = [
  "#8ab4f8",
  "#81c995",
  "#fdd663",
  "#f28b82",
  "#c58af9",
  "#78d9ec",
  "#ff8bcb",
  "#f9ab00",
  "#a8c7fa",
  "#7dd3a8",
];

/** Stable color for a pod name so mixed log streams stay distinguishable. */
export function podColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0;
  }
  return POD_COLORS[hash % POD_COLORS.length];
}
