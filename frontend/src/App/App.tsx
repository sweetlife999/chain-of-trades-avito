import { BrowserRouter, Outlet, Route, Routes } from "react-router-dom";
import styles from "./Styles.module.scss";
import { Header } from "../Components/Pages/Header/Header";

function App() {
  return (
    <>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<OutletWrapper/>}></Route>
        </Routes>
      </BrowserRouter>
    </>
  );
}

function OutletWrapper() {
  return (
    <>
      <Header />
      <Outlet />
    </>
  );
}

// return <div className={styles.counter}>adsasdasj</div>;
export default App;
