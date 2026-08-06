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
import { MyChains } from "../Components/Widgets/MyChainsProcess/MyChains/MyChains";
import { CreateChain } from "../Components/Widgets/MyChainsProcess/CreateChain/CreateChain";
import { AuthInitializer } from "../Components/Widgets/AuthInitializer/AuthInitializer";

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Provider store={store}>
        <BrowserRouter>
        <AuthInitializer>
          <Routes>
            <Route path="/" element={<Layout />}>
              <Route index element={<Main />} />
              <Route path="/login" element={<Login />} />
              <Route path="/register" element={<Register />} />
              <Route path="/profile" element={<Profile />} />
              <Route path="/myChains" element={<MyChains />} />
              <Route path="/exchanges/create" element={<CreateChain />} />
            </Route>
          </Routes>
        </AuthInitializer>
        </BrowserRouter>
      </Provider>
    </QueryClientProvider>
  );
}

export default App;
