package dto

import (
	"time"

	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
)

// Фотография — это ссылка, которую вернул POST /uploads. Внешние http(s)-адреса тоже
// принимаются: с ними живут объявления, созданные до загрузки файлов.
type CreateItemRequest struct {
	Category    string   `json:"category"    example:"bikes"`
	Title       string   `json:"title"       example:"Велосипед"`
	Description string   `json:"description" example:"Почти новый, катался год"`
	PhotoURLs   []string `json:"photo_urls"  example:"/uploads/8db9f3e2-8a45-4a70-b3d1-167b4f97e121.jpg"`
	Wants       []string `json:"wants"       example:"consoles,phones"`
}

// Пропущенный в JSON список приходит как nil, а переданный пустым — как пустой срез.
// На этой разнице держится запрет «удалить все фотографии»: `[]` доедет до сервиса
// и получит 400, а не будет принят за «поле не трогали».
type UpdateItemRequest struct {
	Category    *string  `json:"category"    example:"bikes"`
	Title       *string  `json:"title"       example:"Велосипед городской"`
	Description *string  `json:"description" example:"Добавил фото рамы"`
	PhotoURLs   []string `json:"photo_urls"  example:"/uploads/8db9f3e2-8a45-4a70-b3d1-167b4f97e121.jpg"`
	Wants       []string `json:"wants"       example:"consoles"`
}

type SetPickupPointRequest struct {
	PickupPointID string `json:"pickup_point_id" example:"6f1c1c53-1f0e-4a0a-9d0f-2f0b6c7a1e11"`
}

// Имя не PickupPointResponse: такой тип уже есть в pickuppoint/dto, и swag развёл бы
// коллизию нечитаемыми именами схем — тот же случай, что с ItemError.
type ItemPickupPoint struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
}

type ItemResponse struct {
	ID          string   `json:"id"`
	OwnerID     string   `json:"owner_id"`
	Category    string   `json:"category"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	PhotoURLs   []string `json:"photo_urls"`
	Wants       []string `json:"wants"`
	Status      string   `json:"status"`
	// null — вещь дома у владельца.
	PickupPoint *ItemPickupPoint `json:"pickup_point" extensions:"x-nullable"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

type CategoryResponse struct {
	Slug string `json:"slug" example:"bikes"`
	Name string `json:"name" example:"Велосипеды и транспорт"`
}

// Имя типа не ErrorResponse: у user/dto уже есть такой, и swag разводит коллизию
// именами схем вида github_com_..._internal_item_dto.ErrorResponse.
type ItemError struct {
	Error string `json:"error"`
}

func FromModel(item itemmodel.Item) ItemResponse {
	return ItemResponse{
		ID:          item.ID.String(),
		OwnerID:     item.OwnerID.String(),
		Category:    item.Category,
		Title:       item.Title,
		Description: item.Description,
		PhotoURLs:   item.PhotoURLs,
		Wants:       item.Wants,
		Status:      item.Status,
		PickupPoint: pickupPointFromModel(item.PickupPoint),
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func pickupPointFromModel(point *itemmodel.PickupPoint) *ItemPickupPoint {
	if point == nil {
		return nil
	}

	return &ItemPickupPoint{
		ID:      point.ID.String(),
		Name:    point.Name,
		Address: point.Address,
	}
}

func ItemsFromModel(items []itemmodel.Item) []ItemResponse {
	response := make([]ItemResponse, 0, len(items))
	for _, item := range items {
		response = append(response, FromModel(item))
	}

	return response
}

func CategoriesFromModel(categories []itemmodel.Category) []CategoryResponse {
	response := make([]CategoryResponse, 0, len(categories))
	for _, category := range categories {
		response = append(response, CategoryResponse{Slug: category.Slug, Name: category.Name})
	}

	return response
}
