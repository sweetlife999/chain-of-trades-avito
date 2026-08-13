// components/Layout/Layout.tsx
import { Outlet } from "react-router-dom";
import styles from "./Styles.module.scss";
import { Header } from "../Header/Header";
import { MascotGuide } from "../../UI/MascotGuide/MascotGuide";

const Layout = () => {
  return (
    <>
      <Header />
      <main className={styles.main}>
        <Outlet />
      </main>
      <MascotGuide />
    </>
  );
};



export default Layout;
