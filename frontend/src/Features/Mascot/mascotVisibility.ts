export const shouldShowMascotGuide = (pathname: string) => {
  if (
    pathname === "/" ||
    pathname === "/support" ||
    pathname.startsWith("/admin")
  ) {
    return false;
  }

  // В карточке обмена уже живёт локальный Уми чата: второй экземпляр мешал бы ему.
  return !/^\/exchanges\/[^/]+$/.test(pathname);
};
