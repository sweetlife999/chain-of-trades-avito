import { BrowserRouter, Outlet, Route, Routes } from "react-router-dom";
import Layout from "../Components/Pages/Layout/Layout";
import { Main } from "../Components/Pages/Main/Main";
import { Header } from "../Components/Pages/Header/Header";
import { Login } from "../Components/Pages/Login/Login";
import { Register } from "../Components/Pages/Register/Register";

function App() {
  return (
    <>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<Layout />}>
            <Route index element={<Main />} />
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </>
  );
}

// function OutletWrapper() {
//   return (
//     <>
//       <Header />
//       <main>
//         <Outlet />
//       </main>
//     </>
//   );
// }

export default App;
