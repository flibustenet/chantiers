/*
@copyright  BDL, Bois du Larzac.
@licence    GPL, conformémént au fichier LICENCE situé à la racine du projet.
*/
package control

import (
	"bdl.local/bdl/ctxt"
	"bdl.local/bdl/generic/tiglib"
	"bdl.local/bdl/generic/wilk/webo"
	"bdl.local/bdl/generic/wilk/werr"
	"bdl.local/bdl/model"
	"github.com/gorilla/mux"
	"github.com/jung-kurt/gofpdf"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type detailsChautreForm struct {
	Chantier            *model.Chautre
	TypeChantier        string
	UrlAction           string
	EssenceOptions      template.HTML
	ExploitationOptions template.HTML
	ValorisationOptions template.HTML
	ListeActeurs        map[int]string
	TVAOptions          template.HTML
	AllUGs              []*model.UG
}

type detailsChautreList struct {
	Chantiers []*model.Chautre
	Annee     string   // année courante
	Annees    []string // toutes les années avec chantier
}

type detailsChautreShow struct {
	Chantier *model.Chautre
}

func ListChautre(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	annee := vars["annee"]
	if annee == "" {
		// annee non spécifiée, on prend l'année courante
		annee = strconv.Itoa(time.Now().Year())
	}
	chantiers, err := model.GetChautresOfYear(ctx.DB, annee)
	if err != nil {
		return werr.Wrap(err)
	}
	//
	annees, err := model.GetChautreDifferentYears(ctx.DB, annee)
	if err != nil {
		return werr.Wrap(err)
	}
	//
	titrePage := "Chantiers autres valorisations " + annee
	ctx.TemplateName = "chautre-list.html"
	ctx.Page = &ctxt.Page{
		Header: ctxt.Header{
			Title: titrePage,
			JSFiles: []string{
				"/static/lib/table-sort/table-sort.js",
				"/static/js/round.js",
				"/view/common/prix.js"},
		},
		Menu: "production",
		Details: detailsChautreList{
			Chantiers: chantiers,
			Annee:     annee,
			Annees:    annees,
		},
		Footer: ctxt.Footer{
			JSFiles: []string{
				"/static/lib/table-sort/table-sort.js"},
		},
	}
	return nil
}

// Affichage d'un chantier autres valorisations
func ShowChautre(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	idChantier, err := strconv.Atoi(vars["id"])
	if err != nil {
		return werr.Wrap(err)
	}
	chantier, err := model.GetChautreFull(ctx.DB, idChantier)
	if err != nil {
		return werr.Wrap(err)
	}
	for _, lp := range(chantier.LiensParcelles) {
	    err = lp.Parcelle.ComputeProprietaire(ctx.DB)
        if err != nil {
            return werr.Wrap(err)
        }
	}
	ctx.TemplateName = "chautre-show.html"
	ctx.Page = &ctxt.Page{
		Header: ctxt.Header{
			Title: chantier.FullString(),
			JSFiles: []string{
				"/static/js/round.js",
				"/static/js/formatNb.js",
				"/view/common/prix.js",
			},
		},
		Menu:   "production",
		Footer: ctxt.Footer{},
		Details: detailsChautreShow{
			Chantier: chantier,
		},
	}
	url := r.URL.String()
	err = model.AddRecent(ctx.DB, ctx.Config, &model.Recent{URL: url, Label: chantier.FullString()})
	if err != nil {
		return werr.Wrap(err)
	}
	return nil
}

// Process ou affiche form new
func NewChautre(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "POST":
		//
		// Process form
		//
		chantier, idsUGs, idsLieudits, idsFermiers, err := chautreForm2var(ctx, r)
		if err != nil {
			return werr.Wrap(err)
		}
		//
		chantier.Id, err = model.InsertChautre(ctx.DB, chantier, idsUGs, idsLieudits, idsFermiers)
		if err != nil {
			return werr.Wrap(err)
		}
		ctx.Redirect = "/chantier/autre/" + strconv.Itoa(chantier.Id)
		return nil
	default:
		//
		// Affiche form
		//
		chantier := &model.Chautre{}
		chantier.Acheteur = &model.Acteur{}
		listeActeurs, err := model.GetListeActeurs(ctx.DB)
		if err != nil {
			return werr.Wrap(err)
		}
		allUGs, err := model.GetUGsSortedByCode(ctx.DB)
		if err != nil {
			return werr.Wrap(err)
		}
		ctx.TemplateName = "chautre-form.html"
		ctx.Page = &ctxt.Page{
			Header: ctxt.Header{
				Title: "Nouveau chantier autres valorisations",
				CSSFiles: []string{
					"/static/css/form.css",
					"/static/css/modal.css",
				},
			},
			Menu: "production",
			Footer: ctxt.Footer{
				JSFiles: []string{
					"/static/js/round.js",
					"/static/js/toogle.js",
				},
			},
			Details: detailsChautreForm{
				Chantier:            chantier,
				EssenceOptions:      webo.FmtOptions(WeboEssence(), "CHOOSE_ESSENCE"),
				ExploitationOptions: webo.FmtOptions(WeboExploitation(), "CHOOSE_EXPLOITATION"),
				ValorisationOptions: webo.FmtOptions(WeboChautreValo(), "CHOOSE_VALORISATION"),
				TVAOptions:          webo.FmtOptions(WeboChautreTVA(ctx, "CHOOSE_TVA", "tva-"), "CHOOSE_TVA"),
				ListeActeurs:        listeActeurs,
				AllUGs:              allUGs,
				UrlAction:           "/chantier/autre/new",
			},
		}
		return nil
	}
}

// Process ou affiche form update
func UpdateChautre(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "POST":
		//
		// Process form
		//
		chantier, idsUGs, idsLieudits, idsFermiers, err := chautreForm2var(ctx, r)
		if err != nil {
			return werr.Wrap(err)
		}
		chantier.Id, err = strconv.Atoi(r.PostFormValue("id-chantier"))
		if err != nil {
			return werr.Wrap(err)
		}
		//
		err = model.UpdateChautre(ctx.DB, chantier, idsUGs, idsLieudits, idsFermiers)
		if err != nil {
			return werr.Wrap(err)
		}
		ctx.Redirect = "/chantier/autre/" + strconv.Itoa(chantier.Id)
		// model.AddRecent() inutile puisqu'on est redirigé vers la liste, où AddRecent() est exécuté
		return nil
	default:
		//
		// Affiche form
		//
		vars := mux.Vars(r)
		idChantier, err := strconv.Atoi(vars["id"])
		if err != nil {
			return werr.Wrap(err)
		}
		chantier, err := model.GetChautreFull(ctx.DB, idChantier)
		if err != nil {
			return werr.Wrap(err)
		}
		listeActeurs, err := model.GetListeActeurs(ctx.DB)
		if err != nil {
			return werr.Wrap(err)
		}
		allUGs, err := model.GetUGsSortedByCode(ctx.DB)
		if err != nil {
			return werr.Wrap(err)
		}
		ctx.TemplateName = "chautre-form.html"
		ctx.Page = &ctxt.Page{
			Header: ctxt.Header{
				Title: "Modifier un chantier autres valorisations",
				CSSFiles: []string{
					"/static/css/form.css",
					"/static/css/modal.css"},
			},
			Menu: "production",
			Footer: ctxt.Footer{
				JSFiles: []string{
					"/static/js/round.js", //////////////// a supprimer - dans plaq aussi
					"/static/js/toogle.js"},
			},
			Details: detailsChautreForm{
				Chantier:            chantier,
				TypeChantier:        "chautre",
				EssenceOptions:      webo.FmtOptions(WeboEssence(), "essence-"+chantier.Essence),
				ExploitationOptions: webo.FmtOptions(WeboExploitation(), "exploitation-"+chantier.Exploitation),
				ValorisationOptions: webo.FmtOptions(WeboChautreValo(), "valorisation-"+chantier.TypeValo),
				TVAOptions:          webo.FmtOptions(WeboChautreTVA(ctx, "CHOOSE_TVA", "tva-"), "tva-"+strconv.FormatFloat(chantier.TVA, 'f', -1, 64)),
				ListeActeurs:        listeActeurs,
				AllUGs:              allUGs,
				UrlAction:           "/chantier/autre/update/" + vars["id"],
			},
		}
		// model.AddRecent() inutile puisqu'on est redirigé vers la liste, où AddRecent() est exécuté
		return nil
	}
}

func DeleteChautre(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		return werr.Wrap(err)
	}
	chantier, err := model.GetChautre(ctx.DB, id) // on retient l'année pour le redirect
	if err != nil {
		return werr.Wrap(err)
	}
	err = model.DeleteChautre(ctx.DB, id)
	if err != nil {
		return werr.Wrap(err)
	}
	ctx.Redirect = "/chantier/autre/liste/" + strconv.Itoa(chantier.DateContrat.Year())
	return nil
}

// Fabrique un Chautre à partir des valeurs d'un formulaire.
// Auxiliaire de NewChautre() et UpdateChautre()
// Ne gère pas le champ Id
// Pour form new, IdChantier = 0 ; pour form update, IdChantier a la bonne valeur
// Renvoie idsUG, idsLieudits, idsFermiers car ils ne sont pas stockés dans model.chautre
// Mais les liens avec les parcelles sont stockés dans ch.ChantierParcelle
func chautreForm2var(ctx *ctxt.Context, r *http.Request) (ch *model.Chautre, idsUG, idsLieudits, idsFermiers []int, err error) {
	ch = &model.Chautre{}
	vide := []int{}
	if err = r.ParseForm(); err != nil {
		return ch, vide, vide, vide, werr.Wrap(err)
	}
	//
	ch.Titre = r.PostFormValue("titre")
	//
	idsUG = form2IdsUG(r)
	//
	ch.LiensParcelles = form2LienParcelles(r)
	//
	idsLieudits = form2IdsLieudit(r)
	//
	idsFermiers = form2IdsFermier(r)
	//
	ch.IdAcheteur, err = strconv.Atoi(r.PostFormValue("id-acheteur"))
	if err != nil {
		return ch, vide, vide, vide, werr.Wrap(err)
	}
	//
	ch.TypeVente = strings.Replace(r.PostFormValue("typevente"), "typevente-", "", -1)
	//
	ch.DateContrat, err = time.Parse("2006-01-02", r.PostFormValue("datecontrat"))
	if err != nil {
		return ch, vide, vide, vide, werr.Wrap(err)
	}
	//
	ch.TypeValo = strings.Replace(r.PostFormValue("typevalo"), "valorisation-", "", -1)
	//
	if r.PostFormValue("volume-contrat") == "" {
		ch.VolumeContrat = 0 // car optionnel
	} else {
		ch.VolumeContrat, err = strconv.ParseFloat(r.PostFormValue("volume-contrat"), 32)
		if err != nil {
			return ch, vide, vide, vide, werr.Wrap(err)
		}
	}
	ch.VolumeContrat = tiglib.Round(ch.VolumeContrat, 2)
	//
	ch.VolumeRealise, err = strconv.ParseFloat(r.PostFormValue("volume-realise"), 32)
	if err != nil {
		return ch, vide, vide, vide, werr.Wrap(err)
	}
	ch.VolumeRealise = tiglib.Round(ch.VolumeRealise, 2)
	//
    ch.Unite = model.CodeValo2CodeUnite(ch.TypeValo)
	//
	ch.Exploitation = strings.ReplaceAll(r.PostFormValue("exploitation"), "exploitation-", "")
	//
	ch.Essence = strings.ReplaceAll(r.PostFormValue("essence"), "essence-", "")
	//
	ch.PUHT, err = strconv.ParseFloat(r.PostFormValue("puht"), 32)
	if err != nil {
		return ch, vide, vide, vide, werr.Wrap(err)
	}
	ch.PUHT = tiglib.Round(ch.PUHT, 2)
	//
	ch.TVA, err = strconv.ParseFloat(strings.ReplaceAll(r.PostFormValue("tva"), "tva-", ""), 32)
	if err != nil {
		return ch, vide, vide, vide, werr.Wrap(err)
	}
	//
	if r.PostFormValue("datefacture") != "" {
		ch.DateFacture, err = time.Parse("2006-01-02", r.PostFormValue("datefacture"))
		if err != nil {
			return ch, vide, vide, vide, werr.Wrap(err)
		}
	}
	//
	if r.PostFormValue("datepaiement") != "" {
		ch.DatePaiement, err = time.Parse("2006-01-02", r.PostFormValue("datepaiement"))
		if err != nil {
			return ch, vide, vide, vide, werr.Wrap(err)
		}
	}
	//
	// Création d'un nouveau numéro de facture, uniquement pour form new
	if r.PostFormValue("numfacture") == "" {
		ch.NumFacture, err = model.NouveauNumeroFacture(ctx.DB, strconv.Itoa(ch.DateContrat.Year()))
		if err != nil {
			return ch, vide, vide, vide, werr.Wrap(err)
		}
	} else {
		ch.NumFacture = r.PostFormValue("numfacture")
	}
	//
	ch.Notes = r.PostFormValue("notes")
	//
	return ch, idsUG, idsLieudits, idsFermiers, nil
}

// *********************************************************
func ShowFactureChautre(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		return werr.Wrap(err)
	}
	//
	ch, err := model.GetChautreFull(ctx.DB, id)
	if err != nil {
		return werr.Wrap(err)
	}
	//
	pdf := gofpdf.New("P", "mm", "A4", "")
	InitializeFacture(pdf)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.AddPage()
	//
	MetaDataPDF(pdf, tr, ctx.Config, "Facture bois sur pied")
	HeaderFacture(pdf, tr, ctx.Config)
	FooterFacture(pdf, tr, ctx.Config)
	//
	var str string
	//
	// Acheteur
	//
	pdf.SetXY(60, 70)
	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(100, 7, tr(StringActeurFacture(ch.Acheteur)), "1", "C", false)
	//
	// Date  + n° facture
	//
	var x0, x, y, wi, he float64
	//
	x0 = 10
	x = x0
	y = 110
	wi = 50
	he = 6
	//
	pdf.SetFont("Arial", "B", 10)
	pdf.SetXY(x, y)
	pdf.MultiCell(wi, he, "Date", "1", "C", false)
	//
	x += wi
	pdf.SetXY(x, y)
	pdf.MultiCell(wi, he, tr("Facture n°"), "TRB", "C", false)
	//
	pdf.SetFont("Arial", "", 10)
	x = 10
	y += he
	//
	pdf.SetXY(x, y)
	pdf.MultiCell(wi, he, tiglib.DateFr(ch.DateFacture), "LRB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	pdf.MultiCell(wi, he, ch.NumFacture, "RB", "C", false)
	//
	// Tableau principal
	//
	var w1, w2, w3, w4, w5 = 70.0, 20.0, 20.0, 30.0, 30.0
	x = x0
	y = 140
	pdf.SetXY(x, y)
	wi = w1
	pdf.MultiCell(wi, he, tr("Désignation"), "1", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w2
	pdf.MultiCell(wi, he, tr("Quantité"), "TRB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w3
	pdf.MultiCell(wi, he, tr("Unité"), "TRB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w4
	pdf.MultiCell(wi, he, tr("P.U. € H.T"), "TRB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w5
	pdf.MultiCell(wi, he, tr("Montant € H.T"), "TRB", "C", false)
	//
	x = x0
	y += he
	pdf.SetXY(x, y)
	wi = w1
	str = "Vente " + tr(model.ValoMap[ch.TypeValo]) + " - " + tr(model.EssenceMap[ch.Essence])
	pdf.MultiCell(wi, he, str, "LRB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w2
	pdf.MultiCell(wi, he, strconv.FormatFloat(ch.VolumeRealise, 'f', 2, 64), "RB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w3
	pdf.MultiCell(wi, he, tr(model.UniteMap[ch.Unite]), "RB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w4
	pdf.MultiCell(wi, he, strconv.FormatFloat(ch.PUHT, 'f', 2, 64), "RB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w5
	prixHT := ch.VolumeRealise * ch.PUHT
	pdf.MultiCell(wi, he, strconv.FormatFloat(prixHT, 'f', 2, 64), "RB", "C", false)
	//
	pdf.SetFont("Arial", "B", 10)
	x = x0 + w1
	y += he
	pdf.SetXY(x, y)
	wi = w2 + w3 + w4 + w5
	// @todo arriver à dire euro : € \u20AC
	pdf.MultiCell(wi, he, tr("Montant total € HT"), "RBL", "C", false)
	//
	pdf.SetFont("Arial", "", 10)
	x = x0 + w1
	y += he
	pdf.SetXY(x, y)
	wi = w2 + w3
	pdf.MultiCell(wi, he, "Montant TVA", "RBL", "C", false)
	x += wi
	wi = w4
	pdf.SetXY(x, y)
	pdf.MultiCell(wi, he, strconv.FormatFloat(ch.TVA, 'f', 2, 64)+" %", "RB", "C", false)
	x += wi
	wi = w5
	pdf.SetXY(x, y)
	prixTVA := prixHT * ch.TVA / 100
	pdf.MultiCell(wi, he, strconv.FormatFloat(prixTVA, 'f', 2, 64), "RB", "C", false)
	//
	pdf.SetFont("Arial", "B", 10)
	x = x0 + w1
	y += 2 * he
	pdf.SetXY(x, y)
	wi = w2 + w3 + w4 + w5
	pdf.MultiCell(wi, he, "Montant total TTC", "1", "C", false)
	pdf.SetFont("Arial", "", 10)
	x = x0 + w1
	y += he
	pdf.SetXY(x, y)
	wi = w2 + w3 + w4
	pdf.MultiCell(wi, he, tr("Net à payer en euros"), "RBL", "C", false)
	pdf.SetFont("Arial", "B", 10)
	x += wi
	wi = w5
	pdf.SetXY(x, y)
	prixTTC := prixHT + prixTVA
	pdf.MultiCell(wi, he, strconv.FormatFloat(prixTTC, 'f', 2, 64), "RB", "C", false)
	//
	return pdf.Output(w)
}
