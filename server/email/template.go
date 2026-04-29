package email

import (
	"bytes"
	"html"
	"html/template"
	"strings"
)

// RenderHTML wraps a plain-text body in the MagnetHome brand email shell:
// dark band header (gold mark + wordmark + tagline), white body, dark footer
// with contact/zones/social, all built with table-based markup so it renders
// in Outlook, Apple Mail, Gmail and the rest.
//
// The body's paragraphs (blocks separated by blank lines) become <p>; single
// newlines become <br>. User input is HTML-escaped before paragraph splitting.
func RenderHTML(body string) string {
	var buf bytes.Buffer
	if err := emailTmpl.Execute(&buf, struct {
		Paragraphs []template.HTML
	}{Paragraphs: paragraphs(body)}); err != nil {
		return "<pre>" + html.EscapeString(body) + "</pre>"
	}
	return buf.String()
}

func paragraphs(body string) []template.HTML {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	blocks := strings.Split(strings.TrimSpace(body), "\n\n")
	out := make([]template.HTML, 0, len(blocks))
	for _, b := range blocks {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		escaped := html.EscapeString(b)
		escaped = strings.ReplaceAll(escaped, "\n", "<br>")
		out = append(out, template.HTML(escaped))
	}
	return out
}

var emailTmpl = template.Must(template.New("email").Parse(`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">
<html xmlns="http://www.w3.org/1999/xhtml" lang="es">
<head>
<meta charset="utf-8"/>
<meta name="viewport" content="width=device-width,initial-scale=1"/>
<meta http-equiv="X-UA-Compatible" content="IE=edge"/>
<meta name="x-apple-disable-message-reformatting"/>
<title>MagnetHome</title>
<!--[if mso]>
<noscript><xml><o:OfficeDocumentSettings><o:PixelsPerInch>96</o:PixelsPerInch></o:OfficeDocumentSettings></xml></noscript>
<![endif]-->
<style>
  body,table,td,a{-webkit-text-size-adjust:100%;-ms-text-size-adjust:100%}
  table,td{mso-table-lspace:0pt;mso-table-rspace:0pt}
  img{-ms-interpolation-mode:bicubic;border:0;outline:none;text-decoration:none;display:block}
  body{margin:0;padding:0;width:100%!important;background:#F2F2F0}
  a{color:#A8884B;text-decoration:none}
  @media screen and (max-width:620px){
    .container{width:100%!important;max-width:100%!important}
    .px{padding-left:24px!important;padding-right:24px!important}
    .footer-cols td{display:block!important;width:100%!important;padding:0 0 20px 0!important}
    .tagline{display:none!important}
  }
</style>
</head>
<body style="margin:0;padding:0;background:#F2F2F0;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;">
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" bgcolor="#F2F2F0">
  <tr>
    <td align="center" style="padding:24px 12px;">
      <table role="presentation" class="container" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px;max-width:600px;background:#FFFFFF;">

        <!-- Header: minimal cream — black border-top, centered mark + serif wordmark + gold rule tagline -->
        <tr>
          <td bgcolor="#FFFFFF" style="background:#FFFFFF;padding:0;border-top:3px solid #0D1313;">
            <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
              <tr>
                <td class="px" align="center" style="padding:38px 40px 14px 40px;">
                  <img src="https://magnethome.es/img/email-mark-black.png" width="48" height="48" alt="MagnetHome" style="display:block;margin:0 auto;width:48px;height:48px;"/>
                </td>
              </tr>
              <tr>
                <td class="px" align="center" style="padding:8px 40px 12px 40px;font-family:Georgia,'Times New Roman',serif;font-size:26px;font-weight:400;color:#0D1313;letter-spacing:5px;text-transform:uppercase;">
                  M<span style="font-size:22px;letter-spacing:3.5px;">AGNET</span>H<span style="font-size:22px;letter-spacing:3.5px;">OME</span>
                </td>
              </tr>
              <tr>
                <td class="px" align="center" style="padding:0 40px 32px 40px;">
                  <table role="presentation" cellpadding="0" cellspacing="0" border="0" align="center">
                    <tr>
                      <td style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:10px;color:#A8884B;letter-spacing:3px;text-transform:uppercase;font-weight:600;">
                        <span style="display:inline-block;width:24px;height:1px;background:#CCAA63;vertical-align:middle;margin-right:10px;font-size:0;line-height:0;">&nbsp;</span>
                        Reformas · Costa del Sol
                        <span style="display:inline-block;width:24px;height:1px;background:#CCAA63;vertical-align:middle;margin-left:10px;font-size:0;line-height:0;">&nbsp;</span>
                      </td>
                    </tr>
                  </table>
                </td>
              </tr>
            </table>
          </td>
        </tr>

        <!-- Body -->
        <tr>
          <td bgcolor="#FFFFFF" class="px" style="padding:48px 40px 48px 40px;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:15.5px;line-height:1.7;color:#1A1A1A;">
            {{- range .Paragraphs }}
            <p style="margin:0 0 18px 0;">{{ . }}</p>
            {{- end }}
          </td>
        </tr>

        <!-- Footer -->
        <tr>
          <td bgcolor="#0D1313" style="background:#0D1313;padding:0;">
            <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
              <tr><td style="height:1px;background:#CCAA63;line-height:1px;font-size:0;">&nbsp;</td></tr>
            </table>
            <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
              <tr>
                <td class="px" style="padding:36px 40px 28px 40px;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;">
                  <!-- Lockup -->
                  <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin-bottom:28px;">
                    <tr>
                      <td valign="middle" width="38" style="padding-right:12px;">
                        <img src="https://magnethome.es/img/email-mark.png" width="30" height="30" alt="" style="display:block;width:30px;height:30px;"/>
                      </td>
                      <td valign="middle">
                        <div style="font-family:Georgia,'Times New Roman',serif;font-size:18px;color:#CCAA63;letter-spacing:3px;text-transform:uppercase;font-weight:400;line-height:1.1;">
                          M<span style="font-size:15px;letter-spacing:2.2px;">AGNET</span>H<span style="font-size:15px;letter-spacing:2.2px;">OME</span>
                        </div>
                        <div style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:11px;color:#9A9F9F;margin-top:4px;letter-spacing:0.5px;">
                          Reformas integrales · Costa del Sol
                        </div>
                      </td>
                    </tr>
                  </table>

                  <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0" class="footer-cols">
                    <tr>
                      <td valign="top" width="36%" style="padding:0 12px 0 0;">
                        <div style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:10px;color:#CCAA63;letter-spacing:2.5px;text-transform:uppercase;font-weight:600;margin-bottom:12px;">
                          Contacto
                        </div>
                        <div style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:13px;color:#FFFFFF;line-height:1.8;">
                          <a href="tel:+34643692727" style="color:#FFFFFF;text-decoration:none;">+34 643 692 727</a><br/>
                          <a href="https://wa.me/34643692727" style="color:#FFFFFF;text-decoration:none;">WhatsApp</a><br/>
                          <a href="mailto:info@magnethome.es" style="color:#FFFFFF;text-decoration:none;">info@magnethome.es</a>
                        </div>
                      </td>
                      <td valign="top" width="40%" style="padding:0 12px;">
                        <div style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:10px;color:#CCAA63;letter-spacing:2.5px;text-transform:uppercase;font-weight:600;margin-bottom:12px;">
                          Zonas de servicio
                        </div>
                        <div style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:13px;color:#FFFFFF;line-height:1.8;">
                          Málaga · Marbella · Fuengirola<br/>Benalmádena · Mijas · Ronda
                        </div>
                      </td>
                      <td valign="top" width="24%" style="padding:0 0 0 12px;">
                        <div style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:10px;color:#CCAA63;letter-spacing:2.5px;text-transform:uppercase;font-weight:600;margin-bottom:12px;">
                          Síguenos
                        </div>
                        <a href="https://www.instagram.com/magnethome_/" style="display:inline-block;width:34px;height:34px;border:1px solid #CCAA63;border-radius:50%;text-align:center;line-height:32px;color:#CCAA63;text-decoration:none;font-family:Helvetica,Arial,sans-serif;font-size:11px;font-weight:700;letter-spacing:0;">Ig</a>
                      </td>
                    </tr>
                  </table>

                  <div style="border-top:1px solid rgba(204,170,99,0.18);margin-top:32px;padding-top:20px;">
                    <table role="presentation" width="100%" cellpadding="0" cellspacing="0" border="0">
                      <tr>
                        <td style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:11px;color:#7A7F7F;line-height:1.6;">
                          © 2026 MagnetHome. Todos los derechos reservados.<br/>
                          Costa del Sol, Málaga · España
                        </td>
                        <td align="right" style="font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:11px;color:#CCAA63;letter-spacing:1.5px;">
                          <a href="https://magnethome.es" style="color:#CCAA63;text-decoration:none;">magnethome.es</a>
                        </td>
                      </tr>
                    </table>
                  </div>
                </td>
              </tr>
            </table>
          </td>
        </tr>

      </table>

      <!-- Disclaimer below the card -->
      <table role="presentation" class="container" width="600" cellpadding="0" cellspacing="0" border="0" style="width:600px;max-width:600px;">
        <tr>
          <td class="px" align="center" style="padding:20px 24px 8px 24px;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;font-size:11px;line-height:1.6;color:#9A9A95;">
            Has recibido este correo porque eres cliente o has solicitado información en magnethome.es<br/>
            <a href="https://magnethome.es" style="color:#9A9A95;text-decoration:underline;">magnethome.es</a>
            &nbsp;·&nbsp;
            <a href="https://magnethome.es/aviso-legal.html" style="color:#9A9A95;text-decoration:underline;">Aviso legal</a>
            &nbsp;·&nbsp;
            <a href="https://magnethome.es/privacidad.html" style="color:#9A9A95;text-decoration:underline;">Política de privacidad</a>
          </td>
        </tr>
      </table>

    </td>
  </tr>
</table>
</body>
</html>`))
