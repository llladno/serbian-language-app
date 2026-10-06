// Russian plural forms: 1 пёрышко, 2 пёрышка, 5 пёрышек. The three words come
// from the admin panel (economy_settings), so the app never hardcodes the name
// of the currency — only the rule for picking between the forms.
export function pluralRu(n: number, one: string, few: string, many: string): string {
  const mod100 = Math.abs(n) % 100
  if (mod100 >= 11 && mod100 <= 14) return many
  const mod10 = mod100 % 10
  if (mod10 === 1) return one
  if (mod10 >= 2 && mod10 <= 4) return few
  return many
}
