import { BrowserRouter, Route, Routes } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { Provider } from "react-redux";

import { queryClient } from "../Api/queryClient";
import { store } from "../Store/store";
import Layout from "../Components/Pages/Layout/Layout";
import { Main } from "../Components/Pages/Main/Main";
import { Login } from "../Components/Pages/Login/Login";
import { Register } from "../Components/Pages/Register/Register";
import { Profile } from "../Components/Pages/Profile/Profile";
import { AuthInitializer } from "../Components/Widgets/AuthInitializer/AuthInitializer";
import { MyItems } from "../Components/Widgets/MyItems/MyItems";
import { ItemDetails } from "../Components/Widgets/ItemDetails/ItemDetails";
import { MyChains } from "../Components/Widgets/MyChainsProcess/MyChains/MyChains";
import { CreateChain } from "../Components/Widgets/MyChainsProcess/CreateChain/CreateChain";
import { ExchangeDetails } from "../Components/Widgets/MyChainsProcess/ExchangeDetails/ExchangeDetails";

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Provider store={store}>
        <BrowserRouter>
          <AuthInitializer>
            <Routes>
              <Route path="/" element={<Layout />}>
                <Route index element={<Main />} />
                <Route path="login" element={<Login />} />
                <Route path="register" element={<Register />} />
                <Route path="profile" element={<Profile />} />
                <Route path="myItems" element={<MyItems />} />
                <Route path="items/:id" element={<ItemDetails />} />
                <Route path="exchanges" element={<MyChains />} />
                <Route path="exchanges/create" element={<CreateChain />} />
                <Route path="exchanges/:id" element={<ExchangeDetails />} />
              </Route>
            </Routes>
          </AuthInitializer>
        </BrowserRouter>
      </Provider>
    </QueryClientProvider>
  );
}

export default App;
