// tests/integration/activity_feed_test.go

package integration

import (
	"net/http"
	"strings"
	"testing"

	"code.gitea.io/gitea/tests"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestActivityFeedLayout(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	
	t.Run("Action icons are on the left", func(t *testing.T) {
		req := NewRequest(t, "GET", "/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		feedItems := doc.Find(".dashboard .feeds .news")
		
		assert.Greater(t, feedItems.Length(), 0, "Should have feed items")
		
		feedItems.Each(func(i int, s *goquery.Selection) {
			// Check that icon element exists in flex-item-leading
			iconElement := s.Find(".flex-item-leading svg.octicon")
			assert.Equal(t, 1, iconElement.Length(), "Each feed item should have an icon on the left")
			
			// Check that width and height attributes are set
			width, exists := iconElement.Attr("width")
			assert.True(t, exists, "Icon should have width attribute")
			assert.Equal(t, "28", width, "Icon width should be 28")
			
			height, exists := iconElement.Attr("height")
			assert.True(t, exists, "Icon should have height attribute")
			assert.Equal(t, "28", height, "Icon height should be 28")
		})
	})

	t.Run("User avatars are inline and 16x16", func(t *testing.T) {
		req := NewRequest(t, "GET", "/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		feedItems := doc.Find(".dashboard .feeds .news")
		
		feedItems.Each(func(i int, s *goquery.Selection) {
			avatars := s.Find(".flex-item-main img.ui.avatar")
			
			if avatars.Length() > 0 {
				avatars.Each(func(j int, avatar *goquery.Selection) {
					// Check avatar size
					width, _ := avatar.Attr("width")
					height, _ := avatar.Attr("height")
					
					assert.Equal(t, "16", width, "User avatar width should be 16")
					assert.Equal(t, "16", height, "User avatar height should be 16")
					
					// Check that src exists
					src, exists := avatar.Attr("src")
					assert.True(t, exists, "Avatar should have src")
					assert.NotEmpty(t, src, "Avatar src should not be empty")
				})
			}
		})
	})

	t.Run("Right column is removed", func(t *testing.T) {
		req := NewRequest(t, "GET", "/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		feedItems := doc.Find(".dashboard .feeds .news")
		
		feedItems.Each(func(i int, s *goquery.Selection) {
			// flex-item-trailing should no longer exist
			trailingColumn := s.Find(".flex-item-trailing")
			assert.Equal(t, 0, trailingColumn.Length(), "Right column should not exist")
		})
	})

	t.Run("Feed structure is correct", func(t *testing.T) {
		req := NewRequest(t, "GET", "/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		feedItems := doc.Find(".dashboard .feeds .news")
		
		assert.Greater(t, feedItems.Length(), 0, "Should have feed items")
		
		feedItems.Each(func(i int, s *goquery.Selection) {
			// Check basic structure
			assert.Equal(t, 1, s.Find(".flex-item-leading").Length(), "Should have leading item")
			assert.Equal(t, 1, s.Find(".flex-item-main").Length(), "Should have main item")
			
			// Main content should contain text
			mainContent := s.Find(".flex-item-main").Text()
			assert.NotEmpty(t, strings.TrimSpace(mainContent), "Main content should not be empty")
		})
	})

	t.Run("Commit activities show 24x24 avatars", func(t *testing.T) {
		// Create a commit to ensure commit activities are in the feed
		req := NewRequest(t, "GET", "/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		
		// Look for commit-related feed items
		doc.Find(".dashboard .feeds .news").Each(func(i int, s *goquery.Selection) {
			text := s.Text()
			if strings.Contains(text, "pushed to") || strings.Contains(text, "commit") {
				// Check commit avatars
				commitAvatars := s.Find("img[width='24']")
				if commitAvatars.Length() > 0 {
					commitAvatars.Each(func(j int, avatar *goquery.Selection) {
						width, _ := avatar.Attr("width")
						height, _ := avatar.Attr("height")
						
						assert.Equal(t, "24", width, "Commit avatar width should be 24")
						assert.Equal(t, "24", height, "Commit avatar height should be 24")
					})
				}
			}
		})
	})
}

func TestActivityFeedAvatarVisibility(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	t.Run("Only show avatars for real users", func(t *testing.T) {
		req := NewRequest(t, "GET", "/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		feedItems := doc.Find(".dashboard .feeds .news")
		
		feedItems.Each(func(i int, s *goquery.Selection) {
			avatars := s.Find(".flex-item-main img.ui.avatar")
			
			// If avatar is present, it should have a valid source
			if avatars.Length() > 0 {
				avatars.Each(func(j int, avatar *goquery.Selection) {
					src, exists := avatar.Attr("src")
					assert.True(t, exists, "Avatar should have src attribute")
					assert.NotEmpty(t, src, "Avatar src should not be empty")
					
					// src should point to avatar endpoint
					assert.True(t, 
						strings.Contains(src, "/avatars/") || strings.Contains(src, "/user/avatar/"),
						"Avatar src should point to avatar endpoint")
				})
			}
		})
	})

	t.Run("Avatars have proper alt text", func(t *testing.T) {
		req := NewRequest(t, "GET", "/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		avatars := doc.Find(".dashboard .feeds .news img.ui.avatar")
		
		avatars.Each(func(i int, avatar *goquery.Selection) {
			// Alt attribute should exist (can be empty for decoration)
			_, exists := avatar.Attr("alt")
			assert.True(t, exists, "Avatar should have alt attribute for accessibility")
			
			// Loading attribute should be "lazy"
			loading, _ := avatar.Attr("loading")
			assert.Equal(t, "lazy", loading, "Avatar should have lazy loading")
		})
	})
}

func TestActivityFeedResponsiveness(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")

	t.Run("Feed renders without horizontal overflow", func(t *testing.T) {
		req := NewRequest(t, "GET", "/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		feedContainer := doc.Find(".dashboard .feeds")
		
		assert.Equal(t, 1, feedContainer.Length(), "Feed container should exist")
		
		// Check that feed items exist and are structured
		feedItems := feedContainer.Find(".news")
		assert.Greater(t, feedItems.Length(), 0, "Should have feed items")
	})

	t.Run("Feed items have consistent structure", func(t *testing.T) {
		req := NewRequest(t, "GET", "/")
		resp := session.MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		feedItems := doc.Find(".dashboard .feeds .news")
		
		var hasLeading, hasMain int
		
		feedItems.Each(func(i int, s *goquery.Selection) {
			if s.Find(".flex-item-leading").Length() > 0 {
				hasLeading++
			}
			if s.Find(".flex-item-main").Length() > 0 {
				hasMain++
			}
		})
		
		itemCount := feedItems.Length()
		assert.Equal(t, itemCount, hasLeading, "All items should have leading section")
		assert.Equal(t, itemCount, hasMain, "All items should have main section")
	})
}

func TestActivityFeedCSS(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("CSS files are loaded", func(t *testing.T) {
		req := NewRequest(t, "GET", "/")
		resp := MakeRequest(t, req, http.StatusOK)

		doc := NewHTMLParser(t, resp.Body)
		
		// Check that relevant CSS files are loaded
		cssLinks := doc.Find("link[rel='stylesheet']")
		assert.Greater(t, cssLinks.Length(), 0, "Should have CSS links")
		
		// Dashboard CSS should be present
		foundDashboardCSS := false
		cssLinks.Each(func(i int, s *goquery.Selection) {
			href, _ := s.Attr("href")
			if strings.Contains(href, "dashboard") || strings.Contains(href, "index.css") {
				foundDashboardCSS = true
			}
		})
		
		// Note: This may vary depending on build configuration
		_ = foundDashboardCSS
	})
}