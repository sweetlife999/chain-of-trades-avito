const gradients = [
  "linear-gradient(135deg, aqua, #6a11cb)",
  "linear-gradient(135deg, #ff6b6b, #feca57)",
  "linear-gradient(135deg, #48c6ef, #6f86d6)",
  "linear-gradient(135deg, #f093fb, #f5576c)",
  "linear-gradient(135deg, #4facfe, #00f2fe)",
  "linear-gradient(135deg, #e5d1ff, #a878ff)",
];

export const getAvatarGradient = (id: string) => {
  const hash = id
    .split("")
    .reduce((value, symbol) => {
      return value + symbol.charCodeAt(0);
    }, 0);

  return gradients[hash % gradients.length];
};