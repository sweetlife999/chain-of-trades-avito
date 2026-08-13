export const shouldShowMascotGuide = (pathname: string) => {
  if (
    pathname === "/" ||
    pathname === "/support" ||
    pathname.startsWith("/admin")
  ) {
    return false;
  }

  // На странице обмена проводник не появляется поверх обычного чата.
  return !/^\/exchanges\/[^/]+$/.test(pathname);
};
