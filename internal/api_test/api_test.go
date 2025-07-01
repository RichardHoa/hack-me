package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/RichardHoa/hack-me/internal/app"
	"github.com/RichardHoa/hack-me/internal/routes"
)

func cleanDB(db *sql.DB) {
	db.Exec(`TRUNCATE TABLE "user" RESTART IDENTITY CASCADE`)
	// Repeat for other tables
}

func makeRequestAndExpectStatus(t *testing.T, client *http.Client, method, urlStr string, payload map[string]string, expectedStatus int) []byte {
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(method, urlStr, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// Check for CSRF token in cookies
	if client.Jar != nil {
		u, err := url.Parse(urlStr)
		if err == nil {
			for _, cookie := range client.Jar.Cookies(u) {
				t.Logf("cookie name: %v", cookie.Name)
				if cookie.Name == "csrfToken" {
					req.Header.Set("X-CSRF-Token", cookie.Value)
					break
				}
			}
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	respBody, _ := io.ReadAll(resp.Body)
	t.Logf("status: %d, body: %s", resp.StatusCode, string(respBody))
	resp.Body.Close()

	if resp.StatusCode != expectedStatus {
		t.Errorf("Expected status %d, got %d", expectedStatus, resp.StatusCode)
	}

	return respBody
}

type testRequest struct {
	method string
	path   string
	body   map[string]string
}

type testStep struct {
	name         string
	request      testRequest
	expectStatus int
	validate     func(t *testing.T, body []byte)
}

func TestUserSignUp(t *testing.T) {
	application, err := app.NewApplication(true)
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	defer application.DB.Close()
	defer cleanDB(application.DB)

	router := routes.SetUpRoutes(application)
	server := httptest.NewServer(router)
	defer server.Close()

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
	}

	tests := []struct {
		name           string
		payload        map[string]string
		expectedStatus int
	}{
		{
			name: "Sign up with password-Success",
			payload: map[string]string{
				"userName":  "Richard Hoa",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "testEmail@gmail.com",
				"imageLink": "example.image.com",
				"googleID":  "",
				"githubID":  "",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Duplicate email and username-Failure",
			payload: map[string]string{
				"userName":  "Richard Hoa",
				"password":  "ThisIsAVerySEcurePasswordThatWon'tBeStop",
				"email":     "testEmail@gmail.com",
				"imageLink": "example.image.com",
				"googleID":  "",
				"githubID":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Weak password-Failure",
			payload: map[string]string{
				"userName":  "AnotherUser",
				"password":  "HelloThere",
				"email":     "another@gmail.com",
				"imageLink": "img.com",
				"googleID":  "",
				"githubID":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Lacking password-Failure",
			payload: map[string]string{
				"userName":  "fourth user",
				"password":  "",
				"email":     "anothertest@gmail.com",
				"imageLink": "img.com",
				"googleID":  "",
				"githubID":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Lacking email-Failure",
			payload: map[string]string{
				"userName":  "lacking_email_user_1",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "",
				"imageLink": "",
				"googleID":  "",
				"githubID":  "github-uid-very-unique",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Lacking username-Failure",
			payload: map[string]string{
				"userName":  "",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "lackingusername@gmail.com",
				"imageLink": "",
				"googleID":  "",
				"githubID":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Password and Google ID-Success",
			payload: map[string]string{
				"userName":  "pwg_user",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "pwg_user@gmail.com",
				"imageLink": "",
				"googleID":  "google-uid-123",
				"githubID":  "",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Password and Github ID-Success",
			payload: map[string]string{
				"userName":  "pwh_user",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "pwh_user@gmail.com",
				"imageLink": "",
				"googleID":  "",
				"githubID":  "github-uid-321",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Google ID and GitHub ID-Success",
			payload: map[string]string{
				"userName":  "gg_user",
				"password":  "",
				"email":     "gg_user@gmail.com",
				"imageLink": "",
				"googleID":  "google-uid-456",
				"githubID":  "github-uid-654",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Password, google ID and Github ID-success",
			payload: map[string]string{
				"userName":  "full_user",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "full_user@gmail.com",
				"imageLink": "",
				"googleID":  "google-uid-789",
				"githubID":  "github-uid-987",
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "Duplicate username-Failure",
			payload: map[string]string{
				"userName":  "full_user",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "dup_user1@gmail.com",
				"imageLink": "",
				"googleID":  "",
				"githubID":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate email-Failure",
			payload: map[string]string{
				"userName":  "dup_user2",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "full_user@gmail.com", // same email as above
				"imageLink": "",
				"googleID":  "",
				"githubID":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate Google ID-Failure",
			payload: map[string]string{
				"userName":  "dup_user3",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "dup_google@gmail.com",
				"imageLink": "",
				"googleID":  "google-uid-789", // same as above
				"githubID":  "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate GitHub ID-Failure",
			payload: map[string]string{
				"userName":  "dup_user4",
				"password":  "StrongSecurePasswordThatWon'tBemarkAsInvalid",
				"email":     "dup_github@gmail.com",
				"imageLink": "",
				"googleID":  "",
				"githubID":  "github-uid-987", // same as above
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("%02d-%s", i+1, tc.name), func(t *testing.T) {
			makeRequestAndExpectStatus(t, client, "POST", server.URL+"/v1/users", tc.payload, tc.expectedStatus)
		})
	}
}

func TestChallengeWorkflow(t *testing.T) {
	// Initialize application and server
	application, err := app.NewApplication(true)
	if err != nil {
		t.Fatalf("failed to create application: %v", err)
	}
	defer application.DB.Close()
	defer cleanDB(application.DB)

	router := routes.SetUpRoutes(application)
	server := httptest.NewServer(router)
	defer server.Close()

	// Create persistent client with cookie jar outside the test loop
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Test cases
	tests := []struct {
		name  string
		steps []testStep
	}{
		{
			name: "",
			steps: []testStep{
				{
					name: "No auth tokens",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Vulnaribilities number 1",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusUnauthorized,
				},
				{
					name: "Sign up valid user",
					request: testRequest{
						method: "POST",
						path:   "/v1/users",
						body: map[string]string{
							"userName":  "Richard Hoa",
							"password":  "ThisIsAVerySEcurePasswordThatWon'tBeStop",
							"email":     "testEmail@gmail.com",
							"imageLink": "example.image.com",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "Login test user",
					request: testRequest{
						method: "POST",
						path:   "/v1/users/login",
						body: map[string]string{
							"email":    "testEmail@gmail.com",
							"password": "ThisIsAVerySEcurePasswordThatWon'tBeStop",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "challenge name less than 3 character",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "hs",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "challenge name with leading white space",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "       lots of white space in there",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "challenge name with trailing whitespace",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "lots of white space in there                       ",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "challenge with both white space",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "                           lots of white space in there                       ",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "challenge name with white space only",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     " ",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "no challenge name",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "no challenge content",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Valid name here",
							"content":  "",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusBadRequest,
				},

				{
					name: "no challenge category",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Valid name here",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "challenge with realistic conetent",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name": "Realistically long content",
							"content": `
								# Poenamque bis quantum caput tutaeque rerum

								## Unam sub stabat Marte

								Lorem *markdownum vincula* quam, pollice creditur sciret Iovis: pariter, et
								raptu amplexu memorabat **virum**. Inpellit ossibus transferre Adoni et dignus,
								lapides inpetus paupertatemque supernum ore; aequoreae.

									 if (snmpCellEbook + 5 < 2) {
										  port = layoutDrag;
									 } else {
										  homeComputer = python;
										  operating -= ieeeMountainNetbios;
									 }
									 webcamAnimated = webcamIscsiEmoticon.multi(guiSoft);
									 var constant = apple_rw_scareware;
									 boot_reimage_box(volumeConsoleHsf, vci, function);

								Corpore cum, quod vale et olor adpropera calentes
								[pectore](http://bisque-sed.io/) pulsa suo arbore inquit, sine magna milite
								voluistis. Sanguine volubile fameque: mutua [ultro
								tuum](http://ferorredigar.net/ambiguis-medius.html) quidquid iuvat cum
								invictumque solutus sub reparet dicta, [longoque](http://et.org/).

									 redundancyNocIsp = 6 - sample_dbms_resources.bridgeFile.pebibyte_web_dial(
												web_active);
									 if (xpCore(srgbDesktop) <= ip_webmaster_pup - mirrored_gigahertz) {
										  laptopMetalHtml = script_virus_click - 2;
										  google_compiler.gif_twitter_shell.yobibyte(unicodeRgbIntegrated);
									 } else {
										  hypertext_real_ray = 745726;
									 }
									 if (network_gigabit_mouse.thermistor(denialFull, honeypot_vpn_grep, token)
												== 3) {
										  multitasking = plugDns;
										  commercialComponentIllegal = -1 + 13 - hoverRaidBurn + mysql;
									 } else {
										  volumeMedia.ospf.wRemoteSan(sqlCrossSerp, personal);
									 }

								## Gestu aequoris simul videritis adire

								Illa constitit manibusque mihi et coniunx, fratrisque in obstas iussorum multi,
								et tuum fumida opposita ferrumque. Tam sanguinis, opacae enim rauco ignibus
								Ismariis mando *si corpus* regebat [iuventae de
								senem](http://ligatis-somno.net/). Nisi clara!

								> Videbat pudet gnatae, evomit lues: lucemque et coniunx opus. Namque quae potui
								> tacuit suum eat se clamatque Saturnia, silva. Reccidit castra sic ab Iliades
								> dammis tempore dumque, nec ego des divesque multi.

								Videt corripit de humo vehit rudis poenas Rex, nisi utile dentibus
								[flava](http://nuncsui.com/) loquentem fisso; duo subiere, ille! Non illi haec
								lunae dolorque conditaque tunc, spinea nato vox et glaciali castique.

								Prospicit erit crimina Amphitryoniaden thalamo. Interdum vixque, torrem passa
								omnia fateri dea fragiles, sidera alter ponti omnia, **loqui**. Clara sibi
								aversa posse dixerat vulnera lapides oblitus.

								Fuerat bisque aevi petens Idaeo acta breve simulac centum thalamos [salus
								Marte](http://aut.net/) agebatur recolligat de tremula circum. Diversas
								coniunxque orabam explorant fertur corpus silvestribus profuit nostrumque
								nutrimen excidit ratione carmina, et. Loci adit gerat, sponte si memorque
								perenni insula avidae, posset, per. [Audet vates
								cantibus](http://buxo.net/domos) et, carinae ferrum adflatuque adspexit imas.
								# Poenamque bis quantum caput tutaeque rerum

								## Unam sub stabat Marte

								Lorem *markdownum vincula* quam, pollice creditur sciret Iovis: pariter, et
								raptu amplexu memorabat **virum**. Inpellit ossibus transferre Adoni et dignus,
								lapides inpetus paupertatemque supernum ore; aequoreae.

									 if (snmpCellEbook + 5 < 2) {
										  port = layoutDrag;
									 } else {
										  homeComputer = python;
										  operating -= ieeeMountainNetbios;
									 }
									 webcamAnimated = webcamIscsiEmoticon.multi(guiSoft);
									 var constant = apple_rw_scareware;
									 boot_reimage_box(volumeConsoleHsf, vci, function);

								Corpore cum, quod vale et olor adpropera calentes
								[pectore](http://bisque-sed.io/) pulsa suo arbore inquit, sine magna milite
								voluistis. Sanguine volubile fameque: mutua [ultro
								tuum](http://ferorredigar.net/ambiguis-medius.html) quidquid iuvat cum
								invictumque solutus sub reparet dicta, [longoque](http://et.org/).

									 redundancyNocIsp = 6 - sample_dbms_resources.bridgeFile.pebibyte_web_dial(
												web_active);
									 if (xpCore(srgbDesktop) <= ip_webmaster_pup - mirrored_gigahertz) {
										  laptopMetalHtml = script_virus_click - 2;
										  google_compiler.gif_twitter_shell.yobibyte(unicodeRgbIntegrated);
									 } else {
										  hypertext_real_ray = 745726;
									 }
									 if (network_gigabit_mouse.thermistor(denialFull, honeypot_vpn_grep, token)
												== 3) {
										  multitasking = plugDns;
										  commercialComponentIllegal = -1 + 13 - hoverRaidBurn + mysql;
									 } else {
										  volumeMedia.ospf.wRemoteSan(sqlCrossSerp, personal);
									 }

								## Gestu aequoris simul videritis adire

								Illa constitit manibusque mihi et coniunx, fratrisque in obstas iussorum multi,
								et tuum fumida opposita ferrumque. Tam sanguinis, opacae enim rauco ignibus
								Ismariis mando *si corpus* regebat [iuventae de
								senem](http://ligatis-somno.net/). Nisi clara!

								> Videbat pudet gnatae, evomit lues: lucemque et coniunx opus. Namque quae potui
								> tacuit suum eat se clamatque Saturnia, silva. Reccidit castra sic ab Iliades
								> dammis tempore dumque, nec ego des divesque multi.

								Videt corripit de humo vehit rudis poenas Rex, nisi utile dentibus
								[flava](http://nuncsui.com/) loquentem fisso; duo subiere, ille! Non illi haec
								lunae dolorque conditaque tunc, spinea nato vox et glaciali castique.

								Prospicit erit crimina Amphitryoniaden thalamo. Interdum vixque, torrem passa
								omnia fateri dea fragiles, sidera alter ponti omnia, **loqui**. Clara sibi
								aversa posse dixerat vulnera lapides oblitus.

								Fuerat bisque aevi petens Idaeo acta breve simulac centum thalamos [salus
								Marte](http://aut.net/) agebatur recolligat de tremula circum. Diversas
								coniunxque orabam explorant fertur corpus silvestribus profuit nostrumque
								nutrimen excidit ratione carmina, et. Loci adit gerat, sponte si memorque
								perenni insula avidae, posset, per. [Audet vates
								cantibus](http://buxo.net/domos) et, carinae ferrum adflatuque adspexit imas.
								# Poenamque bis quantum caput tutaeque rerum

								## Unam sub stabat Marte

								Lorem *markdownum vincula* quam, pollice creditur sciret Iovis: pariter, et
								raptu amplexu memorabat **virum**. Inpellit ossibus transferre Adoni et dignus,
								lapides inpetus paupertatemque supernum ore; aequoreae.

									 if (snmpCellEbook + 5 < 2) {
										  port = layoutDrag;
									 } else {
										  homeComputer = python;
										  operating -= ieeeMountainNetbios;
									 }
									 webcamAnimated = webcamIscsiEmoticon.multi(guiSoft);
									 var constant = apple_rw_scareware;
									 boot_reimage_box(volumeConsoleHsf, vci, function);

								Corpore cum, quod vale et olor adpropera calentes
								[pectore](http://bisque-sed.io/) pulsa suo arbore inquit, sine magna milite
								voluistis. Sanguine volubile fameque: mutua [ultro
								tuum](http://ferorredigar.net/ambiguis-medius.html) quidquid iuvat cum
								invictumque solutus sub reparet dicta, [longoque](http://et.org/).

									 redundancyNocIsp = 6 - sample_dbms_resources.bridgeFile.pebibyte_web_dial(
												web_active);
									 if (xpCore(srgbDesktop) <= ip_webmaster_pup - mirrored_gigahertz) {
										  laptopMetalHtml = script_virus_click - 2;
										  google_compiler.gif_twitter_shell.yobibyte(unicodeRgbIntegrated);
									 } else {
										  hypertext_real_ray = 745726;
									 }
									 if (network_gigabit_mouse.thermistor(denialFull, honeypot_vpn_grep, token)
												== 3) {
										  multitasking = plugDns;
										  commercialComponentIllegal = -1 + 13 - hoverRaidBurn + mysql;
									 } else {
										  volumeMedia.ospf.wRemoteSan(sqlCrossSerp, personal);
									 }

								## Gestu aequoris simul videritis adire

								Illa constitit manibusque mihi et coniunx, fratrisque in obstas iussorum multi,
								et tuum fumida opposita ferrumque. Tam sanguinis, opacae enim rauco ignibus
								Ismariis mando *si corpus* regebat [iuventae de
								senem](http://ligatis-somno.net/). Nisi clara!

								> Videbat pudet gnatae, evomit lues: lucemque et coniunx opus. Namque quae potui
								> tacuit suum eat se clamatque Saturnia, silva. Reccidit castra sic ab Iliades
								> dammis tempore dumque, nec ego des divesque multi.

								Videt corripit de humo vehit rudis poenas Rex, nisi utile dentibus
								[flava](http://nuncsui.com/) loquentem fisso; duo subiere, ille! Non illi haec
								lunae dolorque conditaque tunc, spinea nato vox et glaciali castique.

								Prospicit erit crimina Amphitryoniaden thalamo. Interdum vixque, torrem passa
								omnia fateri dea fragiles, sidera alter ponti omnia, **loqui**. Clara sibi
								aversa posse dixerat vulnera lapides oblitus.

								Fuerat bisque aevi petens Idaeo acta breve simulac centum thalamos [salus
								Marte](http://aut.net/) agebatur recolligat de tremula circum. Diversas
								coniunxque orabam explorant fertur corpus silvestribus profuit nostrumque
								nutrimen excidit ratione carmina, et. Loci adit gerat, sponte si memorque
								perenni insula avidae, posset, per. [Audet vates
								cantibus](http://buxo.net/domos) et, carinae ferrum adflatuque adspexit imas.
								# Poenamque bis quantum caput tutaeque rerum

								## Unam sub stabat Marte

								Lorem *markdownum vincula* quam, pollice creditur sciret Iovis: pariter, et
								raptu amplexu memorabat **virum**. Inpellit ossibus transferre Adoni et dignus,
								lapides inpetus paupertatemque supernum ore; aequoreae.

									 if (snmpCellEbook + 5 < 2) {
										  port = layoutDrag;
									 } else {
										  homeComputer = python;
										  operating -= ieeeMountainNetbios;
									 }
									 webcamAnimated = webcamIscsiEmoticon.multi(guiSoft);
									 var constant = apple_rw_scareware;
									 boot_reimage_box(volumeConsoleHsf, vci, function);

								Corpore cum, quod vale et olor adpropera calentes
								[pectore](http://bisque-sed.io/) pulsa suo arbore inquit, sine magna milite
								voluistis. Sanguine volubile fameque: mutua [ultro
								tuum](http://ferorredigar.net/ambiguis-medius.html) quidquid iuvat cum
								invictumque solutus sub reparet dicta, [longoque](http://et.org/).

									 redundancyNocIsp = 6 - sample_dbms_resources.bridgeFile.pebibyte_web_dial(
												web_active);
									 if (xpCore(srgbDesktop) <= ip_webmaster_pup - mirrored_gigahertz) {
										  laptopMetalHtml = script_virus_click - 2;
										  google_compiler.gif_twitter_shell.yobibyte(unicodeRgbIntegrated);
									 } else {
										  hypertext_real_ray = 745726;
									 }
									 if (network_gigabit_mouse.thermistor(denialFull, honeypot_vpn_grep, token)
												== 3) {
										  multitasking = plugDns;
										  commercialComponentIllegal = -1 + 13 - hoverRaidBurn + mysql;
									 } else {
										  volumeMedia.ospf.wRemoteSan(sqlCrossSerp, personal);
									 }

								## Gestu aequoris simul videritis adire

								Illa constitit manibusque mihi et coniunx, fratrisque in obstas iussorum multi,
								et tuum fumida opposita ferrumque. Tam sanguinis, opacae enim rauco ignibus
								Ismariis mando *si corpus* regebat [iuventae de
								senem](http://ligatis-somno.net/). Nisi clara!

								> Videbat pudet gnatae, evomit lues: lucemque et coniunx opus. Namque quae potui
								> tacuit suum eat se clamatque Saturnia, silva. Reccidit castra sic ab Iliades
								> dammis tempore dumque, nec ego des divesque multi.

								Videt corripit de humo vehit rudis poenas Rex, nisi utile dentibus
								[flava](http://nuncsui.com/) loquentem fisso; duo subiere, ille! Non illi haec
								lunae dolorque conditaque tunc, spinea nato vox et glaciali castique.

								Prospicit erit crimina Amphitryoniaden thalamo. Interdum vixque, torrem passa
								omnia fateri dea fragiles, sidera alter ponti omnia, **loqui**. Clara sibi
								aversa posse dixerat vulnera lapides oblitus.

								Fuerat bisque aevi petens Idaeo acta breve simulac centum thalamos [salus
								Marte](http://aut.net/) agebatur recolligat de tremula circum. Diversas
								coniunxque orabam explorant fertur corpus silvestribus profuit nostrumque
								nutrimen excidit ratione carmina, et. Loci adit gerat, sponte si memorque
								perenni insula avidae, posset, per. [Audet vates
								cantibus](http://buxo.net/domos) et, carinae ferrum adflatuque adspexit imas.
								# Poenamque bis quantum caput tutaeque rerum

								## Unam sub stabat Marte

								Lorem *markdownum vincula* quam, pollice creditur sciret Iovis: pariter, et
								raptu amplexu memorabat **virum**. Inpellit ossibus transferre Adoni et dignus,
								lapides inpetus paupertatemque supernum ore; aequoreae.

									 if (snmpCellEbook + 5 < 2) {
										  port = layoutDrag;
									 } else {
										  homeComputer = python;
										  operating -= ieeeMountainNetbios;
									 }
									 webcamAnimated = webcamIscsiEmoticon.multi(guiSoft);
									 var constant = apple_rw_scareware;
									 boot_reimage_box(volumeConsoleHsf, vci, function);

								Corpore cum, quod vale et olor adpropera calentes
								[pectore](http://bisque-sed.io/) pulsa suo arbore inquit, sine magna milite
								voluistis. Sanguine volubile fameque: mutua [ultro
								tuum](http://ferorredigar.net/ambiguis-medius.html) quidquid iuvat cum
								invictumque solutus sub reparet dicta, [longoque](http://et.org/).

									 redundancyNocIsp = 6 - sample_dbms_resources.bridgeFile.pebibyte_web_dial(
												web_active);
									 if (xpCore(srgbDesktop) <= ip_webmaster_pup - mirrored_gigahertz) {
										  laptopMetalHtml = script_virus_click - 2;
										  google_compiler.gif_twitter_shell.yobibyte(unicodeRgbIntegrated);
									 } else {
										  hypertext_real_ray = 745726;
									 }
									 if (network_gigabit_mouse.thermistor(denialFull, honeypot_vpn_grep, token)
												== 3) {
										  multitasking = plugDns;
										  commercialComponentIllegal = -1 + 13 - hoverRaidBurn + mysql;
									 } else {
										  volumeMedia.ospf.wRemoteSan(sqlCrossSerp, personal);
									 }

								## Gestu aequoris simul videritis adire

								Illa constitit manibusque mihi et coniunx, fratrisque in obstas iussorum multi,
								et tuum fumida opposita ferrumque. Tam sanguinis, opacae enim rauco ignibus
								Ismariis mando *si corpus* regebat [iuventae de
								senem](http://ligatis-somno.net/). Nisi clara!

								> Videbat pudet gnatae, evomit lues: lucemque et coniunx opus. Namque quae potui
								> tacuit suum eat se clamatque Saturnia, silva. Reccidit castra sic ab Iliades
								> dammis tempore dumque, nec ego des divesque multi.

								Videt corripit de humo vehit rudis poenas Rex, nisi utile dentibus
								[flava](http://nuncsui.com/) loquentem fisso; duo subiere, ille! Non illi haec
								lunae dolorque conditaque tunc, spinea nato vox et glaciali castique.

								Prospicit erit crimina Amphitryoniaden thalamo. Interdum vixque, torrem passa
								omnia fateri dea fragiles, sidera alter ponti omnia, **loqui**. Clara sibi
								aversa posse dixerat vulnera lapides oblitus.

								Fuerat bisque aevi petens Idaeo acta breve simulac centum thalamos [salus
								Marte](http://aut.net/) agebatur recolligat de tremula circum. Diversas
								coniunxque orabam explorant fertur corpus silvestribus profuit nostrumque
								nutrimen excidit ratione carmina, et. Loci adit gerat, sponte si memorque
								perenni insula avidae, posset, per. [Audet vates
								cantibus](http://buxo.net/domos) et, carinae ferrum adflatuque adspexit imas.
								# Poenamque bis quantum caput tutaeque rerum

								## Unam sub stabat Marte

								Lorem *markdownum vincula* quam, pollice creditur sciret Iovis: pariter, et
								raptu amplexu memorabat **virum**. Inpellit ossibus transferre Adoni et dignus,
								lapides inpetus paupertatemque supernum ore; aequoreae.

									 if (snmpCellEbook + 5 < 2) {
										  port = layoutDrag;
									 } else {
										  homeComputer = python;
										  operating -= ieeeMountainNetbios;
									 }
									 webcamAnimated = webcamIscsiEmoticon.multi(guiSoft);
									 var constant = apple_rw_scareware;
									 boot_reimage_box(volumeConsoleHsf, vci, function);

								Corpore cum, quod vale et olor adpropera calentes
								[pectore](http://bisque-sed.io/) pulsa suo arbore inquit, sine magna milite
								voluistis. Sanguine volubile fameque: mutua [ultro
								tuum](http://ferorredigar.net/ambiguis-medius.html) quidquid iuvat cum
								invictumque solutus sub reparet dicta, [longoque](http://et.org/).

									 redundancyNocIsp = 6 - sample_dbms_resources.bridgeFile.pebibyte_web_dial(
												web_active);
									 if (xpCore(srgbDesktop) <= ip_webmaster_pup - mirrored_gigahertz) {
										  laptopMetalHtml = script_virus_click - 2;
										  google_compiler.gif_twitter_shell.yobibyte(unicodeRgbIntegrated);
									 } else {
										  hypertext_real_ray = 745726;
									 }
									 if (network_gigabit_mouse.thermistor(denialFull, honeypot_vpn_grep, token)
												== 3) {
										  multitasking = plugDns;
										  commercialComponentIllegal = -1 + 13 - hoverRaidBurn + mysql;
									 } else {
										  volumeMedia.ospf.wRemoteSan(sqlCrossSerp, personal);
									 }

								## Gestu aequoris simul videritis adire

								Illa constitit manibusque mihi et coniunx, fratrisque in obstas iussorum multi,
								et tuum fumida opposita ferrumque. Tam sanguinis, opacae enim rauco ignibus
								Ismariis mando *si corpus* regebat [iuventae de
								senem](http://ligatis-somno.net/). Nisi clara!

								> Videbat pudet gnatae, evomit lues: lucemque et coniunx opus. Namque quae potui
								> tacuit suum eat se clamatque Saturnia, silva. Reccidit castra sic ab Iliades
								> dammis tempore dumque, nec ego des divesque multi.

								Videt corripit de humo vehit rudis poenas Rex, nisi utile dentibus
								[flava](http://nuncsui.com/) loquentem fisso; duo subiere, ille! Non illi haec
								lunae dolorque conditaque tunc, spinea nato vox et glaciali castique.

								Prospicit erit crimina Amphitryoniaden thalamo. Interdum vixque, torrem passa
								omnia fateri dea fragiles, sidera alter ponti omnia, **loqui**. Clara sibi
								aversa posse dixerat vulnera lapides oblitus.

								Fuerat bisque aevi petens Idaeo acta breve simulac centum thalamos [salus
								Marte](http://aut.net/) agebatur recolligat de tremula circum. Diversas
								coniunxque orabam explorant fertur corpus silvestribus profuit nostrumque
								nutrimen excidit ratione carmina, et. Loci adit gerat, sponte si memorque
								perenni insula avidae, posset, per. [Audet vates
								cantibus](http://buxo.net/domos) et, carinae ferrum adflatuque adspexit imas.
								# Poenamque bis quantum caput tutaeque rerum

								## Unam sub stabat Marte

								Lorem *markdownum vincula* quam, pollice creditur sciret Iovis: pariter, et
								raptu amplexu memorabat **virum**. Inpellit ossibus transferre Adoni et dignus,
								lapides inpetus paupertatemque supernum ore; aequoreae.

									 if (snmpCellEbook + 5 < 2) {
										  port = layoutDrag;
									 } else {
										  homeComputer = python;
										  operating -= ieeeMountainNetbios;
									 }
									 webcamAnimated = webcamIscsiEmoticon.multi(guiSoft);
									 var constant = apple_rw_scareware;
									 boot_reimage_box(volumeConsoleHsf, vci, function);

								Corpore cum, quod vale et olor adpropera calentes
								[pectore](http://bisque-sed.io/) pulsa suo arbore inquit, sine magna milite
								voluistis. Sanguine volubile fameque: mutua [ultro
								tuum](http://ferorredigar.net/ambiguis-medius.html) quidquid iuvat cum
								invictumque solutus sub reparet dicta, [longoque](http://et.org/).

									 redundancyNocIsp = 6 - sample_dbms_resources.bridgeFile.pebibyte_web_dial(
												web_active);
									 if (xpCore(srgbDesktop) <= ip_webmaster_pup - mirrored_gigahertz) {
										  laptopMetalHtml = script_virus_click - 2;
										  google_compiler.gif_twitter_shell.yobibyte(unicodeRgbIntegrated);
									 } else {
										  hypertext_real_ray = 745726;
									 }
									 if (network_gigabit_mouse.thermistor(denialFull, honeypot_vpn_grep, token)
												== 3) {
										  multitasking = plugDns;
										  commercialComponentIllegal = -1 + 13 - hoverRaidBurn + mysql;
									 } else {
										  volumeMedia.ospf.wRemoteSan(sqlCrossSerp, personal);
									 }

								## Gestu aequoris simul videritis adire

								Illa constitit manibusque mihi et coniunx, fratrisque in obstas iussorum multi,
								et tuum fumida opposita ferrumque. Tam sanguinis, opacae enim rauco ignibus
								Ismariis mando *si corpus* regebat [iuventae de
								senem](http://ligatis-somno.net/). Nisi clara!

								> Videbat pudet gnatae, evomit lues: lucemque et coniunx opus. Namque quae potui
								> tacuit suum eat se clamatque Saturnia, silva. Reccidit castra sic ab Iliades
								> dammis tempore dumque, nec ego des divesque multi.

								Videt corripit de humo vehit rudis poenas Rex, nisi utile dentibus
								[flava](http://nuncsui.com/) loquentem fisso; duo subiere, ille! Non illi haec
								lunae dolorque conditaque tunc, spinea nato vox et glaciali castique.

								Prospicit erit crimina Amphitryoniaden thalamo. Interdum vixque, torrem passa
								omnia fateri dea fragiles, sidera alter ponti omnia, **loqui**. Clara sibi
								aversa posse dixerat vulnera lapides oblitus.

								Fuerat bisque aevi petens Idaeo acta breve simulac centum thalamos [salus
								Marte](http://aut.net/) agebatur recolligat de tremula circum. Diversas
								coniunxque orabam explorant fertur corpus silvestribus profuit nostrumque
								nutrimen excidit ratione carmina, et. Loci adit gerat, sponte si memorque
								perenni insula avidae, posset, per. [Audet vates
								cantibus](http://buxo.net/domos) et, carinae ferrum adflatuque adspexit imas.
								# Poenamque bis quantum caput tutaeque rerum

								## Unam sub stabat Marte

								Lorem *markdownum vincula* quam, pollice creditur sciret Iovis: pariter, et
								raptu amplexu memorabat **virum**. Inpellit ossibus transferre Adoni et dignus,
								lapides inpetus paupertatemque supernum ore; aequoreae.

									 if (snmpCellEbook + 5 < 2) {
										  port = layoutDrag;
									 } else {
										  homeComputer = python;
										  operating -= ieeeMountainNetbios;
									 }
									 webcamAnimated = webcamIscsiEmoticon.multi(guiSoft);
									 var constant = apple_rw_scareware;
									 boot_reimage_box(volumeConsoleHsf, vci, function);

								Corpore cum, quod vale et olor adpropera calentes
								[pectore](http://bisque-sed.io/) pulsa suo arbore inquit, sine magna milite
								voluistis. Sanguine volubile fameque: mutua [ultro
								tuum](http://ferorredigar.net/ambiguis-medius.html) quidquid iuvat cum
								invictumque solutus sub reparet dicta, [longoque](http://et.org/).

									 redundancyNocIsp = 6 - sample_dbms_resources.bridgeFile.pebibyte_web_dial(
												web_active);
									 if (xpCore(srgbDesktop) <= ip_webmaster_pup - mirrored_gigahertz) {
										  laptopMetalHtml = script_virus_click - 2;
										  google_compiler.gif_twitter_shell.yobibyte(unicodeRgbIntegrated);
									 } else {
										  hypertext_real_ray = 745726;
									 }
									 if (network_gigabit_mouse.thermistor(denialFull, honeypot_vpn_grep, token)
												== 3) {
										  multitasking = plugDns;
										  commercialComponentIllegal = -1 + 13 - hoverRaidBurn + mysql;
									 } else {
										  volumeMedia.ospf.wRemoteSan(sqlCrossSerp, personal);
									 }

								## Gestu aequoris simul videritis adire

								Illa constitit manibusque mihi et coniunx, fratrisque in obstas iussorum multi,
								et tuum fumida opposita ferrumque. Tam sanguinis, opacae enim rauco ignibus
								Ismariis mando *si corpus* regebat [iuventae de
								senem](http://ligatis-somno.net/). Nisi clara!

								> Videbat pudet gnatae, evomit lues: lucemque et coniunx opus. Namque quae potui
								> tacuit suum eat se clamatque Saturnia, silva. Reccidit castra sic ab Iliades
								> dammis tempore dumque, nec ego des divesque multi.

								Videt corripit de humo vehit rudis poenas Rex, nisi utile dentibus
								[flava](http://nuncsui.com/) loquentem fisso; duo subiere, ille! Non illi haec
								lunae dolorque conditaque tunc, spinea nato vox et glaciali castique.

								Prospicit erit crimina Amphitryoniaden thalamo. Interdum vixque, torrem passa
								omnia fateri dea fragiles, sidera alter ponti omnia, **loqui**. Clara sibi
								aversa posse dixerat vulnera lapides oblitus.

								Fuerat bisque aevi petens Idaeo acta breve simulac centum thalamos [salus
								Marte](http://aut.net/) agebatur recolligat de tremula circum. Diversas
								coniunxque orabam explorant fertur corpus silvestribus profuit nostrumque
								nutrimen excidit ratione carmina, et. Loci adit gerat, sponte si memorque
								perenni insula avidae, posset, per. [Audet vates
								cantibus](http://buxo.net/domos) et, carinae ferrum adflatuque adspexit imas.
								# Poenamque bis quantum caput tutaeque rerum

								## Unam sub stabat Marte

								Lorem *markdownum vincula* quam, pollice creditur sciret Iovis: pariter, et
								raptu amplexu memorabat **virum**. Inpellit ossibus transferre Adoni et dignus,
								lapides inpetus paupertatemque supernum ore; aequoreae.

									 if (snmpCellEbook + 5 < 2) {
										  port = layoutDrag;
									 } else {
										  homeComputer = python;
										  operating -= ieeeMountainNetbios;
									 }
									 webcamAnimated = webcamIscsiEmoticon.multi(guiSoft);
									 var constant = apple_rw_scareware;
									 boot_reimage_box(volumeConsoleHsf, vci, function);

								Corpore cum, quod vale et olor adpropera calentes
								[pectore](http://bisque-sed.io/) pulsa suo arbore inquit, sine magna milite
								voluistis. Sanguine volubile fameque: mutua [ultro
								tuum](http://ferorredigar.net/ambiguis-medius.html) quidquid iuvat cum
								invictumque solutus sub reparet dicta, [longoque](http://et.org/).

									 redundancyNocIsp = 6 - sample_dbms_resources.bridgeFile.pebibyte_web_dial(
												web_active);
									 if (xpCore(srgbDesktop) <= ip_webmaster_pup - mirrored_gigahertz) {
										  laptopMetalHtml = script_virus_click - 2;
										  google_compiler.gif_twitter_shell.yobibyte(unicodeRgbIntegrated);
									 } else {
										  hypertext_real_ray = 745726;
									 }
									 if (network_gigabit_mouse.thermistor(denialFull, honeypot_vpn_grep, token)
												== 3) {
										  multitasking = plugDns;
										  commercialComponentIllegal = -1 + 13 - hoverRaidBurn + mysql;
									 } else {
										  volumeMedia.ospf.wRemoteSan(sqlCrossSerp, personal);
									 }

								## Gestu aequoris simul videritis adire

								Illa constitit manibusque mihi et coniunx, fratrisque in obstas iussorum multi,
								et tuum fumida opposita ferrumque. Tam sanguinis, opacae enim rauco ignibus
								Ismariis mando *si corpus* regebat [iuventae de
								senem](http://ligatis-somno.net/). Nisi clara!

								> Videbat pudet gnatae, evomit lues: lucemque et coniunx opus. Namque quae potui
								> tacuit suum eat se clamatque Saturnia, silva. Reccidit castra sic ab Iliades
								> dammis tempore dumque, nec ego des divesque multi.

								Videt corripit de humo vehit rudis poenas Rex, nisi utile dentibus
								[flava](http://nuncsui.com/) loquentem fisso; duo subiere, ille! Non illi haec
								lunae dolorque conditaque tunc, spinea nato vox et glaciali castique.

								Prospicit erit crimina Amphitryoniaden thalamo. Interdum vixque, torrem passa
								omnia fateri dea fragiles, sidera alter ponti omnia, **loqui**. Clara sibi
								aversa posse dixerat vulnera lapides oblitus.

								Fuerat bisque aevi petens Idaeo acta breve simulac centum thalamos [salus
								Marte](http://aut.net/) agebatur recolligat de tremula circum. Diversas
								coniunxque orabam explorant fertur corpus silvestribus profuit nostrumque
								nutrimen excidit ratione carmina, et. Loci adit gerat, sponte si memorque
								perenni insula avidae, posset, per. [Audet vates
								cantibus](http://buxo.net/domos) et, carinae ferrum adflatuque adspexit imas.
							`,
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Vulnaribilities number 1",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge with emoji name",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "🚀挑战🔥",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Vulnaribilities number 2",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Vulnaribilities number 3",
							"content":  "This is a very powerful challenge that no one will be able to defeat",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Crackme v1",
							"content":  "Reverse this binary",
							"category": "reverse engineering",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Weak RSA",
							"content":  "Break this RSA",
							"category": "crypto challenge",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 1",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},

				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 2",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 3",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},

				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 4",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 5",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 6",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 7",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 9",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 10",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},

				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Deleted Files 11",
							"content":  "Recover the document",
							"category": "forensics",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "SQL Injection",
							"content":  "Bypass login",
							"category": "web hacking",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "IoT Backdoor",
							"content":  "Find the backdoor",
							"category": "embedded hacking",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "new valid challenge",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name":     "Anti-Debug",
							"content":  "Bypass protections",
							"category": "reverse engineering",
						},
					},
					expectStatus: http.StatusCreated,
				},
				{
					name: "duplicate challenge name",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name": "Vulnaribilities number 1",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "duplicate challenge name(case-insensitive)",
					request: testRequest{
						method: "POST",
						path:   "/v1/challenges",
						body: map[string]string{
							"name": "VulNaRiBiliTiES nUmBer 1",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Modify challenge",
					request: testRequest{
						method: "PUT",
						path:   "/v1/challenges",
						body: map[string]string{
							"oldName":  "Deleted Files 2",
							"name":     "New-Deleted-Files-2",
							"content":  "New content for first modify challenge",
							"category": "crypto challenge",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify the challenge has been modified",
					request: testRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("New-Deleted-Files-2")),
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}
						expectedName := "New-Deleted-Files-2"
						expectedCategory := "crypto challenge"
						expectedUser := "Richard Hoa"
						expectedContent := "New content for first modify challenge"

						found := false
						for _, item := range data {
							challenge, ok := item.(map[string]any)
							if !ok {
								continue
							}

							if challenge["name"] == expectedName &&
								challenge["category"] == expectedCategory &&
								challenge["userName"] == expectedUser &&
								challenge["content"] == expectedContent {
								found = true
								break
							}
						}

						if !found {
							t.Errorf("Expected challenge not found: name=%q, category=%q, userName=%q", expectedName, expectedCategory, expectedUser)
						}
					},
				},
				{
					name: "Modify challenge name",
					request: testRequest{
						method: "PUT",
						path:   "/v1/challenges",
						body: map[string]string{
							"oldName": "New-Deleted-Files-2",
							"name":    "Updated-Name-Only",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify the challenge has been modified",
					request: testRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Updated-Name-Only")),
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}
						data := parsed["data"].([]any)
						if len(data) == 0 {
							t.Fatalf("No challenges returned")
						}
						challenge := data[0].(map[string]any)
						if challenge["name"] != "Updated-Name-Only" {
							t.Errorf("Expected name 'Updated-Name-Only', got %q", challenge["name"])
						}
					},
				},
				{
					name: "Modify challenge content",
					request: testRequest{
						method: "PUT",
						path:   "/v1/challenges",
						body: map[string]string{
							"oldName": "Updated-Name-Only",
							"content": "Updated content only",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify the challenge has been modified",
					request: testRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Updated-Name-Only")),
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}
						data := parsed["data"].([]any)
						if len(data) == 0 {
							t.Fatalf("No challenges returned")
						}
						challenge := data[0].(map[string]any)
						if challenge["content"] != "Updated content only" {
							t.Errorf("Expected content 'Updated content only', got %q", challenge["content"])
						}
					},
				},
				{
					name: "Modify challenge category",
					request: testRequest{
						method: "PUT",
						path:   "/v1/challenges",
						body: map[string]string{
							"oldName":  "Updated-Name-Only",
							"category": "embedded hacking",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify the challenge has been modified",
					request: testRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Updated-Name-Only")),
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}
						data := parsed["data"].([]any)
						if len(data) == 0 {
							t.Fatalf("No challenges returned")
						}
						challenge := data[0].(map[string]any)
						if challenge["category"] != "embedded hacking" {
							t.Errorf("Expected category 'embedded hacking', got %q", challenge["category"])
						}
					},
				},
				{
					name: "Modify challenge user does not own",
					request: testRequest{
						method: "PUT",
						path:   "/v1/challenges",
						body: map[string]string{
							// this old name does not belong to current user
							"oldName": "XSS Lab",
							"name":    "New name",
						},
					},
					expectStatus: http.StatusUnauthorized,
				},
				{
					name: "Modify challenge name that does not eixst",
					request: testRequest{
						method: "PUT",
						path:   "/v1/challenges",
						body: map[string]string{
							"oldName":  "Updated-Name-Only",
							"name":     "IoT Backdoor",
							"category": "",
							"content":  "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Modify challenge name that does not eixst(case-insensitive)",
					request: testRequest{
						method: "PUT",
						path:   "/v1/challenges",
						body: map[string]string{
							"oldName":  "updated-naMe-only",
							"name":     "IoT Backdoor",
							"category": "",
							"content":  "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "No paramter",
					request: testRequest{
						method: "PUT",
						path:   "/v1/challenges",
						body: map[string]string{
							// this old name is valid
							"oldName":  "Updated-Name-Only",
							"name":     "",
							"category": "",
							"content":  "",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "no oldName",
					request: testRequest{
						method: "PUT",
						path:   "/v1/challenges",
						body: map[string]string{
							"oldName":  "",
							"name":     "New name",
							"category": "new content",
							"content":  "new category",
						},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Exact name search",
					request: testRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Vulnaribilities number 1")),
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}
						expectedName := "Vulnaribilities number 1"
						expectedCategory := "web hacking"
						expectedUser := "Richard Hoa"
						expectedContent := "This is a very powerful challenge that no one will be able to defeat"

						found := false
						for _, item := range data {
							challenge, ok := item.(map[string]any)
							if !ok {
								continue
							}

							if challenge["name"] == expectedName &&
								challenge["category"] == expectedCategory &&
								challenge["userName"] == expectedUser &&
								challenge["content"] == expectedContent {
								found = true
								break
							}
						}

						if !found {
							t.Errorf("Expected challenge not found: name=%q, category=%q, userName=%q", expectedName, expectedCategory, expectedUser)
						}
					},
				},
				{
					name: "Generic name search",
					request: testRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?name=%s", url.QueryEscape("Vulnaribilities number")),
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						if len(data) != 3 {
							t.Fatalf("Expected 3 element but we get %v", data)
						}

					},
				},
				{
					name: "Query valid category",
					request: testRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?category=%s", url.QueryEscape("reverse engineering")),
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						if len(data) != 2 {
							t.Fatalf("Expected 2 element but we get %v", len(data))
						}

					},
				},
				{
					name: "Query invalid category",
					request: testRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?category=%s", url.QueryEscape("invalid category")),
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data := parsed["data"]

						if data != nil {
							t.Fatalf("Expected nil data but we get %v", data)
						}

					},
				},

				{
					name: "Query even pageSize",
					request: testRequest{
						method: "GET",
						path:   "/v1/challenges?pageSize=2",
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						if len(data) != 2 {
							t.Fatalf("Expect 2 data response but get %v", len(data))

						}

						metadata, ok := parsed["metadata"].(map[string]interface{})
						if !ok {
							t.Fatalf("Expected metadata to be a map, got: %T", parsed["metadata"])
						}

						if metadata["pageSize"] != "2" || metadata["currentPage"] != "1" || metadata["maxPage"] != "10" {
							t.Fatalf("Expected metadata {pageSize:2, currentPage:1, maxPage:10}, got: %v", metadata)
						}

					},
				},
				{
					name: "Query even pageSize and specific page",
					request: testRequest{
						method: "GET",
						path:   "/v1/challenges?pageSize=2&page=2",
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						if len(data) != 2 {
							t.Fatalf("Expect 2 data response but get %v", len(data))

						}

						metadata, ok := parsed["metadata"].(map[string]interface{})
						if !ok {
							t.Fatalf("Expected metadata to be a map, got: %T", parsed["metadata"])
						}

						if metadata["pageSize"] != "2" || metadata["currentPage"] != "2" || metadata["maxPage"] != "10" {
							t.Fatalf("Expected metadata {pageSize:2, currentPage:2, maxPage:10}, got: %v", metadata)
						}

					},
				},
				{
					name: "Query odd pageSize",
					request: testRequest{
						method: "GET",
						path:   "/v1/challenges?pageSize=3",
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						if len(data) != 3 {
							t.Fatalf("Expect 3 data response but get %v", len(data))

						}

						metadata, ok := parsed["metadata"].(map[string]interface{})
						if !ok {
							t.Fatalf("Expected metadata to be a map, got: %T", parsed["metadata"])
						}

						if metadata["pageSize"] != "3" || metadata["currentPage"] != "1" || metadata["maxPage"] != "7" {
							t.Fatalf("Expected metadata {pageSize:3, currentPage:1, maxPage:7}, got: %v", metadata)
						}

					},
				},
				{
					name: "Query odd pageSize and page",
					request: testRequest{
						method: "GET",
						path:   "/v1/challenges?pageSize=3&page=2",
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}

						data, ok := parsed["data"].([]any)
						if !ok {
							t.Fatalf(`Expected "data" to be a list, got: %#v`, parsed["data"])
						}

						if len(data) != 3 {
							t.Fatalf("Expect 2 data response but get %v", len(data))

						}

						metadata, ok := parsed["metadata"].(map[string]interface{})
						if !ok {
							t.Fatalf("Expected metadata to be a map, got: %T", parsed["metadata"])
						}

						if metadata["pageSize"] != "3" || metadata["currentPage"] != "2" || metadata["maxPage"] != "7" {
							t.Fatalf("Expected metadata {pageSize:3, currentPage:2, maxPage:7}, got: %v", metadata)
						}

					},
				},

				{
					name: "Query negative pageSize",
					request: testRequest{
						method: "GET",
						path:   "/v1/challenges?pageSize=-2",
					},
					expectStatus: http.StatusBadRequest,
				},

				{
					name: "Query 0 pageSize",
					request: testRequest{
						method: "GET",
						path:   "/v1/challenges?pageSize=0",
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Query negative page",
					request: testRequest{
						method: "GET",
						path:   "/v1/challenges?page=-10",
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Query 0 page",
					request: testRequest{
						method: "GET",
						path:   "/v1/challenges?page=0",
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Query invalid page and pagaSize",
					request: testRequest{
						method: "GET",
						path:   "/v1/challenges?page=0&pageSize=0",
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "Delete challenge that user own",
					request: testRequest{
						method: "DELETE",
						path:   "/v1/challenges",
						body: map[string]string{
							"name": "Updated-Name-Only",
						},
					},
					expectStatus: http.StatusOK,
				},
				{
					name: "Verify challenge has been deleted",
					request: testRequest{
						method: "GET",
						path:   fmt.Sprintf("/v1/challenges?exactName=%s", url.QueryEscape("Updated-Name-Only")),
					},
					expectStatus: http.StatusOK,
					validate: func(t *testing.T, body []byte) {
						var parsed map[string]any
						if err := json.Unmarshal(body, &parsed); err != nil {
							t.Fatalf("Failed to parse response: %v", err)
						}
						data := parsed["data"].([]any)
						if len(data) != 0 {
							t.Fatalf("challenge still exist after being deleted %v", data)
						}
					},
				},
				{
					name: "challenge that user does not own",
					request: testRequest{
						method: "DELETE",
						path:   "/v1/challenges",
						body: map[string]string{
							"name": "XSS Lab",
						},
					},
					expectStatus: http.StatusUnauthorized,
				},
				{
					name: "challenge name that does not exist",
					request: testRequest{
						method: "DELETE",
						path:   "/v1/challenges",
						body: map[string]string{
							"name": "random challenge name that definitely do not exist in this test",
						},
					},
					expectStatus: http.StatusUnauthorized,
				},
				{
					name: "no body parameter",
					request: testRequest{
						method: "DELETE",
						path:   "/v1/challenges",
						body:   map[string]string{},
					},
					expectStatus: http.StatusBadRequest,
				},
				{
					name: "challenge name is empty string",
					request: testRequest{
						method: "DELETE",
						path:   "/v1/challenges",
						body:   map[string]string{"name": " "},
					},
					expectStatus: http.StatusUnauthorized,
				},
			},
		},
	}

	test := tests[0]
	// Run test cases
	for _, step := range test.steps {
		t.Run(fmt.Sprintf("%s-%s-%d-%s", step.request.method, step.request.path, step.expectStatus, step.name), func(t *testing.T) {
			body := makeRequestAndExpectStatus(t, client, step.request.method, server.URL+step.request.path, step.request.body, step.expectStatus)

			// Run custom validation if provided
			if step.validate != nil {
				step.validate(t, body)
			}
		})
	}

}
