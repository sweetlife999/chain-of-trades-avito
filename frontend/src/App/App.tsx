import { BrowserRouter, Route, Routes } from "react-router-dom";
import { QueryClientProvider } from "@tanstack/react-query";
import { Provider } from "react-redux";
import { queryClient } from "../Api/queryClient";
import { store } from "../Store/store";
import Layout from "../Components/Pages/Layout/Layout";
import { lazy, Suspense } from "react";
import { Loader } from "../Components/UI/Loader/Loader";
import { AuthInitializer } from "../Components/Widgets/AuthInitializer/AuthInitializer";
// Не lazy: это точка входа на сайт, отдельный чанк здесь стоит лишнего round-trip.
import { Landing } from "../Components/Pages/Landing/Landing";

const Main = lazy(() =>
  import("../Components/Pages/Main/Main").then((module) => ({
    default: module.Main,
  })),
);

const Login = lazy(() =>
  import("../Components/Pages/Login/Login").then((module) => ({
    default: module.Login,
  })),
);

const Register = lazy(() =>
  import("../Components/Pages/Register/Register").then((module) => ({
    default: module.Register,
  })),
);

const Profile = lazy(() =>
  import("../Components/Pages/Profile/Profile").then((module) => ({
    default: module.Profile,
  })),
);

const ProfileEdit = lazy(() =>
  import("../Components/Pages/ProfileEdit/ProfileEdit").then((module) => ({
    default: module.ProfileEdit,
  })),
);

const MyItems = lazy(() =>
  import("../Components/Widgets/MyItems/MyItems").then((module) => ({
    default: module.MyItems,
  })),
);

const ItemDetails = lazy(() =>
  import("../Components/Widgets/ItemDetails/ItemDetails").then((module) => ({
    default: module.ItemDetails,
  })),
);

const ItemEdit = lazy(() =>
  import("../Components/Widgets/ItemEdit/ItemEdit").then((module) => ({
    default: module.ItemEdit,
  })),
);

const MyChains = lazy(() =>
  import("../Components/Widgets/MyChainsProcess/MyChains/MyChains").then(
    (module) => ({
      default: module.MyChains,
    }),
  ),
);

const CreateChain = lazy(() =>
  import("../Components/Widgets/MyChainsProcess/CreateChain/CreateChain").then(
    (module) => ({
      default: module.CreateChain,
    }),
  ),
);

const ExchangeDetails = lazy(() =>
  import("../Components/Widgets/MyChainsProcess/ExchangeDetails/ExchangeDetails").then(
    (module) => ({
      default: module.ExchangeDetails,
    }),
  ),
);

const AdminRoute = lazy(() =>
  import("../Components/Widgets/Admin/AdminRoute/AdminRoute").then(
    (module) => ({
      default: module.AdminRoute,
    }),
  ),
);

const NotificationsPage = lazy(() =>
  import("../Components/Pages/Notifications/NotificationsPage").then(
    (module) => ({
      default: module.NotificationsPage,
    }),
  ),
);

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <Provider store={store}>
        <BrowserRouter>
          <AuthInitializer>
            <Suspense
              fallback={<Loader fullScreen text="Загружаем страницу..." />}
            >
              <Routes>
                <Route path="/" element={<Landing />} />
                <Route element={<Layout />}>
                  <Route path="/feed" element={<Main />} />
                  <Route path="/login" element={<Login />} />
                  <Route path="/register" element={<Register />} />
                  <Route path="/profile" element={<Profile />} />
                  <Route path="/profile/edit" element={<ProfileEdit />} />
                  <Route path="/profile/:id" element={<Profile />} />
                  <Route path="/myItems" element={<MyItems />} />
                  <Route path="/items/:id" element={<ItemDetails />} />
                  <Route path="/items/:id/edit" element={<ItemEdit />} />
                  <Route path="/exchanges" element={<MyChains />} />
                  <Route path="/exchanges/create" element={<CreateChain />} />
                  <Route path="/exchanges/:id" element={<ExchangeDetails />} />
                  <Route path="/notifications" element={<NotificationsPage />} />
                  <Route path="/admin" element={<AdminRoute />} />
                </Route>
              </Routes>
            </Suspense>
          </AuthInitializer>
        </BrowserRouter>
      </Provider>
    </QueryClientProvider>
  );
}

export default App;
