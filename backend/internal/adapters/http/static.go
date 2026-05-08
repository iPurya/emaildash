package httpadapter

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const faviconSVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 64 64" role="img" aria-label="EmailDash">
  <defs>
    <linearGradient id="bg" x1="10" y1="6" x2="54" y2="58" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#152746"/>
      <stop offset="1" stop-color="#070b14"/>
    </linearGradient>
    <linearGradient id="mail" x1="14" y1="17" x2="50" y2="49" gradientUnits="userSpaceOnUse">
      <stop offset="0" stop-color="#69a7ff"/>
      <stop offset="1" stop-color="#2dd4bf"/>
    </linearGradient>
  </defs>
  <rect width="64" height="64" rx="15" fill="url(#bg)"/>
  <path d="M13 22.5A6.5 6.5 0 0 1 19.5 16h25A6.5 6.5 0 0 1 51 22.5v19A6.5 6.5 0 0 1 44.5 48h-25A6.5 6.5 0 0 1 13 41.5v-19Z" fill="#0f1726" stroke="url(#mail)" stroke-width="3"/>
  <path d="m16 22 16 13 16-13" fill="none" stroke="#dbeafe" stroke-width="3" stroke-linecap="round" stroke-linejoin="round" opacity=".9"/>
  <path d="M22 39.5 28.4 34l6 4.2L44 28" fill="none" stroke="#2dd4bf" stroke-width="4" stroke-linecap="round" stroke-linejoin="round"/>
  <circle cx="47" cy="15" r="7" fill="#f6c453"/>
  <circle cx="47" cy="15" r="3" fill="#070b14" opacity=".9"/>
</svg>`

func serveFavicon(c *gin.Context) {
	c.Header("Cache-Control", "public, max-age=604800")
	c.Data(http.StatusOK, "image/svg+xml; charset=utf-8", []byte(faviconSVG))
}

func redirectLegacyFavicon(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "/favicon.svg")
}
