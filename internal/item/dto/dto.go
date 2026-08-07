package dto

import (
	"time"

	itemmodel "github.com/sweetlife999/chain-of-trades-avito/internal/item/model"
)

type CreateItemRequest struct {
	Category    string   `json:"category"    example:"bikes"`
	Title       string   `json:"title"       example:"Велосипед"`
	Description string   `json:"description" example:"Почти новый, катался год"`
	PhotoURLs   []string `json:"photo_urls"  example:"https://example.com/bike-1.jpg,https://example.com/bike-2.jpg"`
	Wants       []string `json:"wants"       example:"consoles,phones"`
}

// Пропущенный в JSON список приходит как nil, а переданный пустым — как пустой срез.
// На этой разнице держится запрет «удалить все фотографии»: `[]` доедет до сервиса
// и получит 400, а не будет принят за «поле не трогали».
type UpdateItemRequest struct {
	Category    *string  `json:"category"    example:"bikes"`
	Title       *string  `json:"title"       example:"Велосипед городской"`
	Description *string  `json:"description" example:"Добавил фото рамы"`
	PhotoURLs   []string `json:"photo_urls"  example:"https://example.com/bike-1.jpg,https://example.com/bike-3.jpg"`
	Wants       []string `json:"wants"       example:"consoles"`
}

type ItemResponse struct {
	ID          string    `json:"id"`
	OwnerID     string    `json:"owner_id"`
	Category    string    `json:"category"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	PhotoURLs   []string  `json:"photo_urls"`
	Wants       []string  `json:"wants"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
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
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
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
