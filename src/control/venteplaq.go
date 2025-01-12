/*
Vente plaquettes

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

type detailsVentePlaqForm struct {
	Vente              *model.VentePlaq
	FournisseurOptions template.HTML
	ListeActeurs       map[int]string
	UrlAction          string
}
type detailsVentePlaqList struct {
	Ventes []*model.VentePlaq
	Annee  string   // année courante
	Annees []string // toutes les années avec chantier
}
type detailsVentePlaqShow struct {
	Vente *model.VentePlaq
}

func ListVentePlaq(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	annee := vars["annee"]
	if annee == "" {
		// annee non spécifiée, on prend l'année courante
		annee = strconv.Itoa(time.Now().Year())
	}
	ventes, err := model.GetVentePlaqsOfYear(ctx.DB, annee)
	if err != nil {
		return werr.Wrap(err)
	}
	//
	annees, err := model.GetVentePlaqDifferentYears(ctx.DB, annee)
	if err != nil {
		return werr.Wrap(err)
	}
	ctx.TemplateName = "venteplaq-list.html"
	ctx.Page = &ctxt.Page{
		Header: ctxt.Header{
			Title: "Ventes plaquettes " + annee,
			JSFiles: []string{
				"/static/js/round.js",
				"/static/js/formatNb.js",
				"/view/common/venteplaq.js",
				"/view/common/prix.js"},
		},
		Menu: "ventes",
		Footer: ctxt.Footer{
			JSFiles: []string{
				"/static/lib/table-sort/table-sort.js",
			},
		},
		Details: detailsVentePlaqList{
			Ventes: ventes,
			Annee:  annee,
			Annees: annees,
		},
	}
	return nil
}

func ShowVentePlaq(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	idVente, _ := strconv.Atoi(vars["id-vente"])
	vente, err := model.GetVentePlaqFull(ctx.DB, idVente)
	if err != nil {
		return werr.Wrap(err)
	}
	ctx.TemplateName = "venteplaq-show.html"
	ctx.Page = &ctxt.Page{
		Header: ctxt.Header{
			Title: vente.FullString(),
			JSFiles: []string{
				"/static/js/round.js",
				"/static/js/formatNb.js",
				"/view/common/venteplaq.js",
				"/view/common/prix.js"},
		},
		Menu: "ventes",
		Footer: ctxt.Footer{
			JSFiles: []string{},
		},
		Details: detailsVentePlaqShow{
			Vente: vente,
		},
	}
	err = model.AddRecent(ctx.DB, ctx.Config, &model.Recent{URL: r.URL.String(), Label: vente.FullString()})
	if err != nil {
		return werr.Wrap(err)
	}
	return nil
}

// Process ou affiche form new
func NewVentePlaq(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "POST":
		//
		// Process form
		//
		vente, err := ventePlaqForm2var(ctx, r)
		if err != nil {
			return werr.Wrap(err)
		}
		vente.TVA = ctx.Config.TVABDL.VentePlaquettes
		vente.FactureLivraisonTVA = ctx.Config.TVABDL.Livraison
		idVente, err := model.InsertVentePlaq(ctx.DB, vente)
		if err != nil {
			return werr.Wrap(err)
		}
		ctx.Redirect = "/vente/" + strconv.Itoa(idVente)
		// model.AddRecent() inutile puisqu'on est redirigé vers ShowVentePlaq(), où AddRecent() est exécuté
		return nil
	default:
		//
		// Affiche form
		//
		vente := &model.VentePlaq{}
		vente.Client = &model.Acteur{}
		vente.Fournisseur = &model.Acteur{}
		vente.TVA = ctx.Config.TVABDL.VentePlaquettes
		vente.FactureLivraisonTVA = ctx.Config.TVABDL.Livraison
		listeActeurs, err := model.GetListeActeurs(ctx.DB)
		if err != nil {
			return werr.Wrap(err)
		}
		//
		weboFournisseur, err := WeboFournisseur(ctx)
		if err != nil {
			return werr.Wrap(err)
		}
		//
		ctx.TemplateName = "venteplaq-form.html"
		ctx.Page = &ctxt.Page{
			Header: ctxt.Header{
				Title: "Nouvelle vente de plaquettes",
				CSSFiles: []string{
					"/static/css/form.css"},
			},
			Menu: "ventes",
			Footer: ctxt.Footer{
				JSFiles: []string{
					"/static/js/toogle.js"},
			},
			Details: detailsVentePlaqForm{
				Vente:              vente,
				FournisseurOptions: webo.FmtOptions(weboFournisseur, "CHOOSE_FOURNISSEUR"),
				ListeActeurs:       listeActeurs,
				UrlAction:          "/vente/new",
			},
		}
		return nil
	}
}

// Process ou affiche form update
func UpdateVentePlaq(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "POST":
		//
		// Process form
		//
		vente, err := ventePlaqForm2var(ctx, r)
		if err != nil {
			return werr.Wrap(err)
		}
		vente.Id, err = strconv.Atoi(r.PostFormValue("id-vente"))
		if err != nil {
			return werr.Wrap(err)
		}
		vente.TVA = ctx.Config.TVABDL.VentePlaquettes
		vente.FactureLivraisonTVA = ctx.Config.TVABDL.Livraison
		err = model.UpdateVentePlaq(ctx.DB, vente)
		if err != nil {
			return werr.Wrap(err)
		}
		ctx.Redirect = "/vente/" + r.PostFormValue("id-vente")
		// model.AddRecent() inutile puisqu'on est redirigé vers ShowVentePlaq(), où AddRecent() est exécuté
		return nil
	default:
		//
		// Affiche form
		//
		vars := mux.Vars(r)
		idVente, err := strconv.Atoi(vars["id-vente"])
		if err != nil {
			return werr.Wrap(err)
		}
		//
		vente, err := model.GetVentePlaqFull(ctx.DB, idVente)
		if err != nil {
			return werr.Wrap(err)
		}
		//
		listeActeurs, err := model.GetListeActeurs(ctx.DB)
		if err != nil {
			return werr.Wrap(err)
		}
		//
		weboFournisseur, err := WeboFournisseur(ctx)
		if err != nil {
			return werr.Wrap(err)
		}
		//
		ctx.TemplateName = "venteplaq-form.html"
		ctx.Page = &ctxt.Page{
			Header: ctxt.Header{
				Title: "Modifier la vente : " + vente.String(),
				CSSFiles: []string{
					"/static/css/form.css"},
			},
			Menu: "ventes",
			Footer: ctxt.Footer{
				JSFiles: []string{
					"/static/js/toogle.js"},
			},
			Details: detailsVentePlaqForm{
				FournisseurOptions: webo.FmtOptions(weboFournisseur, "fournisseur-"+strconv.Itoa(vente.IdFournisseur)),
				Vente:              vente,
				ListeActeurs:       listeActeurs,
				UrlAction:          "/vente/update/" + vars["id-vente"],
			},
		}
		return nil
	}
}

func DeleteVentePlaq(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id-vente"])
	if err != nil {
		return werr.Wrap(err)
	}
	vente, err := model.GetVentePlaq(ctx.DB, id) // pour retenir l'année dans le redirect
	if err != nil {
		return werr.Wrap(err)
	}
	err = model.DeleteVentePlaq(ctx.DB, id)
	if err != nil {
		return werr.Wrap(err)
	}
	ctx.Redirect = "/vente/liste/" + strconv.Itoa(vente.DateVente.Year())
	return nil
}

// Fabrique une VentePlaq à partir des valeurs d'un formulaire.
// Auxiliaire de NewVentePlaq() et UpdateVentePlaq()
// Ne gère pas le champ Id
// Ne gère pas les champs TVA et FactureLivraisonTVA (car viennent de la config)
func ventePlaqForm2var(ctx *ctxt.Context, r *http.Request) (*model.VentePlaq, error) {
	vente := &model.VentePlaq{}
	var err error
	if err = r.ParseForm(); err != nil {
		return vente, werr.Wrap(err)
	}
	//
	vente.IdClient, err = strconv.Atoi(r.PostFormValue("id-client"))
	if err != nil {
		return vente, werr.Wrap(err)
	}
	//
	vente.IdFournisseur, err = strconv.Atoi(strings.TrimLeft(r.PostFormValue("fournisseur"), "fournisseur-"))
	if err != nil {
		return vente, werr.Wrap(err)
	}
	//
	vente.PUHT, err = strconv.ParseFloat(r.PostFormValue("puht"), 32)
	if err != nil {
		return vente, werr.Wrap(err)
	}
	vente.PUHT = tiglib.Round(vente.PUHT, 2)
	//
	vente.DateVente, err = time.Parse("2006-01-02", r.PostFormValue("datevente"))
	if err != nil {
		return vente, werr.Wrap(err)
	}
	//
	// Facture
	//
	// Création d'un nouveau numéro de facture, uniquement pour form new
	if r.PostFormValue("numfacture") == "" {
		vente.NumFacture, err = model.NouveauNumeroFacture(ctx.DB, strconv.Itoa(vente.DateVente.Year()))
		if err != nil {
			return vente, werr.Wrap(err)
		}
	} else {
		vente.NumFacture = r.PostFormValue("numfacture")
	}
	//
	if r.PostFormValue("datefacture") != "" {
		vente.DateFacture, err = time.Parse("2006-01-02", r.PostFormValue("datefacture"))
		if err != nil {
			return vente, werr.Wrap(err)
		}
	}
	//
	if r.PostFormValue("datepaiement") != "" {
		vente.DatePaiement, err = time.Parse("2006-01-02", r.PostFormValue("datepaiement"))
		if err != nil {
			return vente, werr.Wrap(err)
		}
	}
	//
	vente.FactureLivraison = false
	if r.PostFormValue("facturelivraison") == "on" {
		vente.FactureLivraison = true
		//
		vente.FactureLivraisonPUHT, err = strconv.ParseFloat(r.PostFormValue("facturelivraisonpuht"), 32)
		if err != nil {
			return vente, werr.Wrap(err)
		}
		vente.FactureLivraisonPUHT = tiglib.Round(vente.FactureLivraisonPUHT, 2)
		vente.FactureLivraisonUnite = r.PostFormValue("facturelivraisonunite") // map ou km -cf commentaire de classe model.VentePlaq
		if r.PostFormValue("facturelivraisonnbkm") != "" {
			vente.FactureLivraisonNbKm, err = strconv.ParseFloat(r.PostFormValue("facturelivraisonnbkm"), 32)
		}
		if err != nil {
			return vente, werr.Wrap(err)
		}
	}
	// sinon
	// - FactureLivraisonPUHT reste à 0
	// - FactureLivraisonUnite reste à ""
	// => ok
	//
	vente.FactureNotes = false
	if r.PostFormValue("facturenotes") == "on" {
		vente.FactureNotes = true
	}
	//
	vente.Notes = r.PostFormValue("notes")
	//
	return vente, nil
}

func ShowFactureVentePlaq(ctx *ctxt.Context, w http.ResponseWriter, r *http.Request) error {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		return werr.Wrap(err)
	}
	//
	vente, err := model.GetVentePlaqFull(ctx.DB, id)
	if err != nil {
		return werr.Wrap(err)
	}
	//
	pdf := gofpdf.New("P", "mm", "A4", "")
	InitializeFacture(pdf)
	tr := pdf.UnicodeTranslatorFromDescriptor("") // "" defaults to "cp1252"
	pdf.AddPage()
	//
	MetaDataPDF(pdf, tr, ctx.Config, "Facture vente plaquettes")
	HeaderFacture(pdf, tr, ctx.Config)
	FooterFacture(pdf, tr, ctx.Config)
	//
	// Client
	//
	pdf.SetXY(60, 70)
	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(100, 7, tr(StringActeurFacture(vente.Client)), "1", "C", false)
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
	pdf.MultiCell(wi, he, "Date de facturation", "1", "C", false)
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
	pdf.MultiCell(wi, he, tiglib.DateFr(vente.DateFacture), "LRB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	pdf.MultiCell(wi, he, tr(vente.NumFacture), "RB", "C", false)
	//
	// Tableau principal
	//
	var w1, w2, w3, w4, w5 = 70.0, 20.0, 20.0, 30.0, 30.0
	//
	// ligne entête des colonnes
	//
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
	// ligne avec valeurs de la vente
	//
	var prixHTPlaquettes, prixHT float64
	pdf.SetFont("Arial", "B", 10)
	x = x0
	y += he
	pdf.SetXY(x, y)
	wi = w1
	pdf.MultiCell(wi, he, tr("Vente de plaquettes forestières"), "LRB", "L", false)
	pdf.SetFont("Arial", "", 10)
	x += wi
	pdf.SetXY(x, y)
	wi = w2
	pdf.MultiCell(wi, he, strconv.FormatFloat(vente.Qte, 'f', 2, 64), "RB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w3
	pdf.MultiCell(wi, he, "MAP", "RB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w4
	pdf.MultiCell(wi, he, strconv.FormatFloat(vente.PUHT, 'f', 2, 64), "RB", "C", false)
	x += wi
	pdf.SetXY(x, y)
	wi = w5
	prixHTPlaquettes = vente.Qte * vente.PUHT
	pdf.MultiCell(wi, he, strconv.FormatFloat(prixHTPlaquettes, 'f', 2, 64), "RB", "C", false)
	prixHT = prixHTPlaquettes
	//
	// ligne avec valeurs de la livraison
	//
	var prixHTLivraison, qteLivraison float64
	var unite string
	if vente.FactureLivraison {
		if vente.FactureLivraisonUnite == "map" {
			unite = "MAP"
			qteLivraison = vente.Qte
		} else {
			unite = "KM"
			qteLivraison = vente.FactureLivraisonNbKm
		}
		pdf.SetFont("Arial", "B", 10)
		x = x0
		y += he
		pdf.SetXY(x, y)
		wi = w1
		pdf.MultiCell(wi, he, tr("Livraison"), "LRB", "L", false)
		pdf.SetFont("Arial", "", 10)
		x += wi
		pdf.SetXY(x, y)
		wi = w2
		pdf.MultiCell(wi, he, strconv.FormatFloat(qteLivraison, 'f', 2, 64), "RB", "C", false)
		x += wi
		pdf.SetXY(x, y)
		wi = w3
		pdf.MultiCell(wi, he, unite, "RB", "C", false)
		x += wi
		pdf.SetXY(x, y)
		wi = w4
		pdf.MultiCell(wi, he, strconv.FormatFloat(vente.FactureLivraisonPUHT, 'f', 2, 64), "RB", "C", false)
		x += wi
		pdf.SetXY(x, y)
		wi = w5
		prixHTLivraison = qteLivraison * vente.FactureLivraisonPUHT
		pdf.MultiCell(wi, he, strconv.FormatFloat(prixHTLivraison, 'f', 2, 64), "RB", "C", false)
		prixHT += prixHTLivraison
	}
	//
	// ligne avec les notes
	//
	if vente.FactureNotes {
		pdf.SetFont("Arial", "", 10)
		x = x0
		y += he
		pdf.SetXY(x, y)
		wi = w1 + w2 + w3 + w4 + w5
		lines := tiglib.LimitLength(tr(vente.Notes), 108) // 108 mesuré empiriquement
		pdf.MultiCell(wi, he, strings.Join(lines, "\n"), "LRB", "L", false)
		y += he * float64(len(lines))
	} else {
		y += he
	}
	//
	// ligne montant total HT
	//
	pdf.SetFont("Arial", "B", 10)
	x = x0 + w1
	pdf.SetXY(x, y)
	wi = w2 + w3 + w4
	pdf.MultiCell(wi, he, tr("Montant total € HT"), "RBL", "C", false)
	pdf.SetFont("Arial", "", 10)
	x = x0 + w1 + w2 + w3 + w4
	wi = w5
	pdf.SetXY(x, y)
	pdf.MultiCell(wi, he, strconv.FormatFloat(prixHT, 'f', 2, 64), "RBL", "C", false)
	//
	// ligne TVA plaquettes
	//
	var prixTVAPlaquettes float64
	pdf.SetFont("Arial", "", 10)
	x = x0 + w1
	y += he
	pdf.SetXY(x, y)
	wi = w2 + w3
	pdf.MultiCell(wi, he, "Montant TVA plaquettes", "RBL", "L", false)
	x += wi
	wi = w4
	pdf.SetXY(x, y)
	pdf.MultiCell(wi, he, strconv.FormatFloat(vente.TVA, 'f', 2, 64)+" %", "RB", "C", false)
	x += wi
	wi = w5
	pdf.SetXY(x, y)
	prixTVAPlaquettes = prixHTPlaquettes * vente.TVA / 100
	pdf.MultiCell(wi, he, strconv.FormatFloat(prixTVAPlaquettes, 'f', 2, 64), "RB", "C", false)
	//
	// ligne TVA livraison
	//
	var prixTVALivraison float64
	if vente.FactureLivraison {
		pdf.SetFont("Arial", "", 10)
		x = x0 + w1
		y += he
		pdf.SetXY(x, y)
		wi = w2 + w3
		pdf.MultiCell(wi, he, "Montant TVA livraison", "RBL", "L", false)
		x += wi
		wi = w4
		pdf.SetXY(x, y)
		pdf.MultiCell(wi, he, strconv.FormatFloat(vente.FactureLivraisonTVA, 'f', 2, 64)+" %", "RB", "C", false)
		x += wi
		wi = w5
		pdf.SetXY(x, y)
		prixTVALivraison = prixHTLivraison * vente.FactureLivraisonTVA / 100
		pdf.MultiCell(wi, he, strconv.FormatFloat(prixTVALivraison, 'f', 2, 64), "RB", "C", false)
	}
	//
	// 2 lignes pour montant total TTC
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
	prixTTC := prixHT + prixTVAPlaquettes + prixTVALivraison // PrixHT inclue déjà prixHTLivraison
	pdf.MultiCell(wi, he, strconv.FormatFloat(prixTTC, 'f', 2, 64), "RB", "C", false)
	//
	// Noms du ou des livreur(s) / conducteur(s)
	//
	pdf.SetFont("Arial", "", 10)
	x = x0
	y += 2 * he
	pdf.SetXY(x, y)
	pdf.Write(he, tr("Livraison effectuée par"))
	y += he
	for _, livraison := range vente.Livraisons {
		pdf.SetXY(x, y)
		if livraison.TypeCout == "G" { // coût global
			pdf.Write(he, "- "+tr(livraison.Livreur.String()))
		} else { // coût détaillé
			pdf.Write(he, "- "+tr(livraison.Conducteur.String()))
		}
        pdf.Write(he, ", le "+tr(tiglib.DateFr(livraison.DateLivre)))
		
		y += he
	}
	//
	return pdf.Output(w)
}
