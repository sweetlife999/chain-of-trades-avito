import { BrowserRouter, Route, Routes } from "react-router-dom";
import Layout from "../Components/Pages/Layout/Layout";
import { Main } from "../Components/Pages/Main/Main";
import { Login } from "../Components/Pages/Login/Login";
import { Register } from "../Components/Pages/Register/Register";
import { Profile } from "../Components/Pages/Profile/Profile";
import { QueryClientProvider } from "@tanstack/react-query";
import { queryClient } from "../Api/queryClient";
import { Provider } from "react-redux";
import { store } from "../Store/store";

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Provider store={store}>
        <BrowserRouter>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<Main />} />
              <Route path="/login" element={<Login />} />
              <Route path="/register" element={<Register />} />
              <Route path="/profile/:id" element={<Profile />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </Provider>
    </QueryClientProvider>
  );
}

export default App;
