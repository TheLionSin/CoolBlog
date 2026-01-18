package helpers

import (
	"fmt"
	"go_blog/config"
	"go_blog/models"
	"go_blog/utils"
)

func GenerateUniqueSlug(title string) (string, error) {
	base := utils.Slugify(title)
	slug := base

	var count int64
	i := 1
	for {
		config.DB.Model(&models.Post{}).Where("slug = ?", slug).Count(&count)
		if count == 0 {
			return slug, nil
		}
		i++
		slug = fmt.Sprintf("%s-%d", slug, i)
	}
}
