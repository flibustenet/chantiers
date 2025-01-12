/*
Unités de gestion

@copyright  BDL, Bois du Larzac.
@licence    GPL, conformémént au fichier LICENCE situé à la racine du projet.
@history    2019-11-14 23:36:13+01:00, Thierry Graff : Creation
*/
package model

import (
	"bdl.local/bdl/generic/tiglib"
	"bdl.local/bdl/generic/wilk/werr"
	"github.com/jmoiron/sqlx"
	"golang.org/x/exp/slices"
	"sort"
	"strconv"
	"strings"
)

type UG struct {
	Id                        int
	Code                      string
	SurfaceSIG                string `db:"surface_sig"`
	CodeTypo                  string `db:"code_typo"`
	Coupe                     string `db:"coupe"`
	AnneeIntervention         string `db:"annee_intervention"`
	PSGSuivant                string `db:"psg_suivant"`
	VolumeStockOuRecouvrement string `db:"volume_stock_ou_recouvrement"`
	IntensitePrelevement      string `db:"intensite_prelevement"`
	AmenagementDivers         string `db:"amenagement_divers"`
	// pas stocké dans la table ug
	Parcelles        []*Parcelle
	Fermiers         []*Fermier
	Communes         []*Commune
	Proprietaires    []*Acteur
	CodesEssence     []string
	Activites        []*Activite
	Recaps           map[string]RecapUG
	SortedRecapYears []string // années contenant de l'activité prise en compte dans Recaps
}

type RecapUG struct {
	Annee            string // YYYY
	Plaquettes       LigneRecapUG
	PateAPapier      LigneRecapUG
	ChauffageFermier LigneRecapUG
	Chauffage        LigneRecapUG
	Palette          LigneRecapUG
	Piquets          LigneRecapUG
	BoisOeuvre       LigneRecapUG
	BoisSurPied      LigneRecapUG
}

type LigneRecapUG struct {
	Quantite         float64
	Superficie       float64
	CoutExploitation float64
	Benefice         float64
}

// ************************ Nom *********************************

func (ug *UG) String() string {
	return ug.Code
}

// ************************ Codes UG *********************************

// Nombres romains utilisés dans le code des UGs
var romans = []string{
	"I",
	"II",
	"III",
	"IV",
	"V",
	"VI",
	"VII",
	"VIII",
	"IX",
	"X",
	"XI",
	"XII",
	"XIII",
	"XIV",
	"XV",
	"XVI",
	"XVII",
	"XVIII",
	"XIX",
}

// Roman2Arab Convertit un nombre romain en nombre arabe.
// Uniquement pour les nombres romains utilisés dans les codes UG
func roman2Arab(roman string) int {
	idx := slices.Index(romans, roman)
	if idx == -1 {
		return -1
	}
	return idx + 1
}

// SortableCode traduit un code UG en string qui peut être numériquement triée.
// ex: XVI-4 est converti en 1604 (= 1600 + 4)
func SortableUGCode(code string) string {
	tmp := strings.Split(code, "-")
	n1 := roman2Arab(tmp[0])
	n2, _ := strconv.Atoi(tmp[1])
	return strconv.FormatInt(int64(100*n1+n2), 10)
}

// ************************ Get one *********************************

// Renvoie une UG à partir de son id.
// Ne contient que les champs de la table ug.
// Les autres champs ne sont pas remplis.
func GetUG(db *sqlx.DB, id int) (ug *UG, err error) {
	ug = &UG{}
	query := "select * from ug where id=$1"
	row := db.QueryRowx(query, id)
	err = row.StructScan(ug)
	if err != nil {
		return ug, werr.Wrapf(err, "Erreur query : "+query)
	}
	return ug, nil
}

// Calcule les champs d'une UG qui ne sont pas stockés en base.
// Mais ne calcule pas les activités - parce que pas fait dans la v1 - mais pourrait être ajouté (?)
// TODO voir si on ne devrait pas directement faire UG.ComputeLieudits, ComputeProprietaire et ComputeCommune
// avec des jointures plutôt que de passer par la table parcelle
func GetUGFull(db *sqlx.DB, id int) (ug *UG, err error) {
	ug, err = GetUG(db, id)
	if err != nil {
		return ug, werr.Wrapf(err, "Erreur appel GetUG()")
	}
	err = ug.ComputeParcelles(db)
	if err != nil {
		return ug, werr.Wrapf(err, "Erreur appel UG.ComputeParcelles()")
	}
	for i, _ := range ug.Parcelles {
		err = ug.Parcelles[i].ComputeLieudits(db)
		if err != nil {
			return ug, werr.Wrapf(err, "Erreur appel Parcelle.ComputeLieudits()")
		}
		err = ug.Parcelles[i].ComputeProprietaire(db)
		if err != nil {
			return ug, werr.Wrapf(err, "Erreur appel Parcelle.ComputeProprietaire()")
		}
		err = ug.Parcelles[i].ComputeCommune(db)
		if err != nil {
			return ug, werr.Wrapf(err, "Erreur appel Parcelle.ComputeCommune()")
		}
	}
	err = ug.ComputeFermiers(db)
	if err != nil {
		return ug, werr.Wrapf(err, "Erreur appel UG.ComputeFermiers()")
	}
	err = ug.ComputeProprietaires(db)
	if err != nil {
		return ug, werr.Wrapf(err, "Erreur appel UG.ComputeProprietaires()")
	}
	err = ug.ComputeEssences(db)
	if err != nil {
		return ug, werr.Wrapf(err, "Erreur appel UG.ComputeEssences()")
	}
	return ug, nil
}

// Renvoie une UG à partir de son code, ou nil si aucune UG ne correspond au code
// Ne contient que les champs de la table ug.
// Les autres champs ne sont pas remplis.
// Utilisé par ajax
func GetUGFromCode(db *sqlx.DB, code string) (*UG, error) {
	ug := UG{}
	query := "select * from ug where code=$1"
	err := db.Get(&ug, query, code)
	if err != nil {
		return nil, nil
	}
	return &ug, nil
}

// ************************ Get many *********************************

// Renvoie des UGs à partir d'un lieu-dit.
// Utilise les parcelles pour faire le lien
// Ne contient que les champs de la table ug.
// Les autres champs ne sont pas remplis.
// Utilisé par ajax
//
// TODO bizarre, pourquoi ne pas écrire avec une jointure ?
func GetUGsFromLieudit(db *sqlx.DB, idLieudit int) (ugs []*UG, err error) {
	ugs = []*UG{}
	// parcelles
	idsParcelles := []int{}
	query := "select id_parcelle from parcelle_lieudit where id_lieudit=$1"
	err = db.Select(&idsParcelles, query, idLieudit)
	if err != nil {
		return ugs, werr.Wrapf(err, "Erreur query : "+query)
	}
	if len(idsParcelles) == 0 {
		return ugs, nil // empty res
	}
	// ids ugs
	strIdsParcelles := tiglib.JoinInt(idsParcelles, ",")
	idsUGs := []int{}
	query = "select distinct id_ug from parcelle_ug where id_parcelle in(" + strIdsParcelles + ")"
	err = db.Select(&idsUGs, query)
	if err != nil {
		return ugs, werr.Wrapf(err, "Erreur query : "+query)
	}
	if len(idsUGs) == 0 {
		return ugs, nil // empty res
	}
	// ugs
	strIdsUGs := tiglib.JoinInt(idsUGs, ",")
	query = "select * from ug where id in(" + strIdsUGs + ") order by code"
	err = db.Select(&ugs, query)
	if err != nil {
		return ugs, werr.Wrapf(err, "Erreur query : "+query)
	}
	return ugs, nil
}

// Renvoie des UGs à partir d'un fermier.
// Utilise les parcelles pour faire le lien
// Ne contient que les champs de la table ug.
// Les autres champs ne sont pas remplis.
// Utilisé par ajax
func GetUGsFromFermier(db *sqlx.DB, idFermier int) (ugs []*UG, err error) {
	ugs = []*UG{}
	query := `
        select * from ug where id in(
            select id_ug from parcelle_ug where id_parcelle in(
                select id_parcelle from parcelle_fermier where id_fermier in(
                    select id from fermier where id=$1
                )
            )
        ) order by code`
	err = db.Select(&ugs, query, idFermier)
	if err != nil {
		return ugs, werr.Wrapf(db.Select(&ugs, query, idFermier), "Erreur query : "+query)
	}
	return ugs, nil
}

// Renvoie les ugs triées par code (nombre romain) et par numéro au sein d'un code (nombres arabes)
// en respectant l'ordre des chiffres romains et arabes.
func GetUGsSortedByCode(db *sqlx.DB) (ugs []*UG, err error) {
	ugs = []*UG{}
	query := `select * from ug`
	err = db.Select(&ugs, query)
	if err != nil {
		return ugs, werr.Wrapf(err, "Erreur query : "+query)
	}
	sort.Slice(ugs, func(i, j int) bool {
		ug1 := ugs[i]
		ug2 := ugs[j]
		code1 := strings.Replace(ug1.Code, ".", "-", -1) // fix typo dans un code (XIX.5)
		tmp1 := strings.Split(code1, "-")
		code2 := strings.Replace(ug2.Code, ".", "-", -1) // fix typo dans un code (XIX.5)
		tmp2 := strings.Split(code2, "-")
		// teste chiffres romains
		idx1 := tiglib.ArraySearch(romans, tmp1[0])
		idx2 := tiglib.ArraySearch(romans, tmp2[0])
		if idx1 < idx2 {
			return true
		}
		if idx1 > idx2 {
			return false
		}
		// idx1 = idx2 - chiffres romains identiques
		n1, _ := strconv.Atoi(tmp1[1])
		n2, _ := strconv.Atoi(tmp2[1])
		return n1 < n2
	})
	return ugs, nil
}

// Renvoie les ugs triées par code (nombre romain) et par numéro au sein d'un code (nombres arabes)
// en respectant l'ordre des chiffres romains et arabes.
// Renvoie un tableau de tableaux d'UGs dont le code commence par le même nombre romain.
// res[0] : ugs avec code commençant par I-
// res[1] : ugs avec code commençant par II-
// etc.
func GetUGsSortedByCodeAndSeparated(db *sqlx.DB) ([][]*UG, error) {
	res := [][]*UG{}
	ugs, err := GetUGsSortedByCode(db)
	if err != nil {
		return res, werr.Wrapf(err, "Erreur appel GetUGsSortedByCode()")
	}
	curRoman := "I"
	cur := []*UG{}
	for _, ug := range ugs {
		code := strings.Replace(ug.Code, ".", "-", -1) // fix typo dans un code (XIX.5)
		roman := ug.Code[:strings.Index(code, "-")]
		if roman == curRoman {
			cur = append(cur, ug)
		} else {
			// nombre romain différent
			curRoman = roman
			res = append(res, cur)
			cur = []*UG{}
		}
	}
	res = append(res, cur)
	return res, nil
}

// ************************** Compute *******************************

func (ug *UG) ComputeParcelles(db *sqlx.DB) (err error) {
	if len(ug.Parcelles) != 0 {
		return nil // déjà calculé
	}
	query := `
	    select * from parcelle where id in(
            select id_parcelle from parcelle_ug where id_ug=$1
        ) order by code`
	return db.Select(&ug.Parcelles, query, ug.Id)
}

func (ug *UG) ComputeFermiers(db *sqlx.DB) (err error) {
	if len(ug.Fermiers) != 0 {
		return nil // déjà calculé
	}
	query := `
        select * from fermier where id in(
            select id_fermier from parcelle_fermier where id_parcelle in(
                select id_parcelle from parcelle_ug where id_ug=$1
            )
        ) order by nom`
	err = db.Select(&ug.Fermiers, query, ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur query : "+query)
	}
	return nil
}

// Pas utilisé par GetUGFull() - mais pourraît l'être
func (ug *UG) ComputeCommunes(db *sqlx.DB) (err error) {
	if len(ug.Communes) != 0 {
		return nil // déjà calculé
	}
	query := `
        select * from commune where id in(
            select id_commune from parcelle where id in(
                select id_parcelle from parcelle_ug where id_ug=$1
            )
        ) order by nom`
	err = db.Select(&ug.Communes, query, ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur query : "+query)
	}
	return nil
}

func (ug *UG) ComputeProprietaires(db *sqlx.DB) (err error) {
	if len(ug.Proprietaires) != 0 {
		return nil // déjà calculé
	}
	query := `
        select * from acteur where id in(
            select id_proprietaire from parcelle where id in(
                select id_parcelle from parcelle_ug where id_ug=$1
            )
        ) order by nom`
	err = db.Select(&ug.Proprietaires, query, ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur query : "+query)
	}
	return nil
}

func (ug *UG) ComputeEssences(db *sqlx.DB) (err error) {
	if len(ug.CodesEssence) != 0 {
		return nil // déjà calculé
	}
	query := `select code_essence from ug_essence where id_ug =$1 order by code_essence`
	err = db.Select(&ug.CodesEssence, query, ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur query : "+query)
	}
	return nil
}

// Pas inclus dans GetUGFull()
func (ug *UG) ComputeActivites(db *sqlx.DB) (err error) {
	// code possible - mais pas optimisé
	//filtres := map[string][]string{"ug":[]string{strconv.Itoa(ug.Id)}}
	//activites, err := model.ComputeActivitesFromFiltres(ctx.DB, filtres)
	var query string
	chantiers1 := []*Plaq{}
	query = "select * from plaq where id in(select id_chantier from chantier_ug where type_chantier='plaq' and id_ug=$1)"
	err = db.Select(&chantiers1, query, ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur query DB : "+query)
	}
	for _, chantier := range chantiers1 {
		tmp, err := plaq2Activite(db, chantier)
		if err != nil {
			return werr.Wrapf(err, "Erreur appel plaq2Activite()")
		}
		ug.Activites = append(ug.Activites, tmp)
	}
	//
	chantiers2 := []*Chautre{}
	query = "select * from chautre where id in(select id_chantier from chantier_ug where type_chantier='chautre' and id_ug=$1)"
	err = db.Select(&chantiers2, query, ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur query DB : "+query)
	}
	for _, chantier := range chantiers2 {
		tmp, err := chautre2Activite(db, chantier)
		if err != nil {
			return werr.Wrapf(err, "Erreur appel chautre2Activite()")
		}
		ug.Activites = append(ug.Activites, tmp)
	}
	//
	chantiers3 := []*Chaufer{}
	query = "select * from chaufer where id in(select id_chantier from chantier_ug where type_chantier='chaufer' and id_ug=$1)"
	err = db.Select(&chantiers3, query, ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur query DB : "+query)
	}
	for _, chantier := range chantiers3 {
		tmp, err := chaufer2Activite(db, chantier)
		if err != nil {
			return werr.Wrapf(err, "Erreur appel chaufer2Activite()")
		}
		ug.Activites = append(ug.Activites, tmp)
	}
	//
	return nil
}

// ************************** Recap *******************************

// Pas inclus dans GetUGFull()
func (ug *UG) ComputeRecap(db *sqlx.DB) error {
	var err error
	ids := []int{}
	ug.Recaps = make(map[string]RecapUG)
	//
	// Chantiers plaquettes
	//
	ids, err = computeIdsChantiersFromUG(db, "plaq", ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur appel computeIdsChantiersFromUG()")
	}
	for _, idChantier := range ids {
		chantier, err := GetPlaqFull(db, idChantier)
		if err != nil {
			return werr.Wrapf(err, "Erreur appel GetPlaqFull()")
		}
		y := strconv.Itoa(chantier.DateDebut.Year())
		myrecap := ug.Recaps[y] // à cause de pb "cannot assign"
		myrecap.Annee = y       // au cas où on l'utilise pour la 1e fois
		myrecap.Plaquettes.Quantite += chantier.Volume
		myrecap.Plaquettes.Superficie += chantier.Surface
		// TODO myrecap.Plaquettes.CoutExploitation
		// TODO myrecap.Plaquettes.Benefice
		ug.Recaps[y] = myrecap
	}
	//
	// Chantiers chauffage fermier
	//
	ids, err = computeIdsChantiersFromUG(db, "chaufer", ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur appel computeIdsChantiersFromUG()")
	}
	for _, idChantier := range ids {
		chantier, err := GetChauferFull(db, idChantier)
		if err != nil {
			return werr.Wrapf(err, "Erreur appel GetChauferFull()")
		}
		y := strconv.Itoa(chantier.DateChantier.Year())
		myrecap := ug.Recaps[y] // à cause de pb "cannot assign"
		myrecap.Annee = y       // au cas où on l'utilise pour la 1e fois
		myrecap.ChauffageFermier.Quantite += chantier.Volume
		// TODO myrecap.ChauffageFermier.Superficie
		myrecap.ChauffageFermier.CoutExploitation = 0
		myrecap.ChauffageFermier.Benefice = 0
		ug.Recaps[y] = myrecap
	}
	//
	// Chantier autres valorisations
	//
	ids, err = computeIdsChantiersFromUG(db, "chautre", ug.Id)
	if err != nil {
		return werr.Wrapf(err, "Erreur appel computeIdsChantiersFromUG()")
	}
	for _, idChantier := range ids {
		chantier, err := GetChautreFull(db, idChantier)
		if err != nil {
			return werr.Wrapf(err, "Erreur appel GetChautreFull()")
		}
		y := strconv.Itoa(chantier.DateContrat.Year())
		myrecap := ug.Recaps[y] // à cause de pb "cannot assign"
		myrecap.Annee = y       // au cas où on l'utilise pour la 1e fois
		switch chantier.TypeValo {
		case "BO":
			myrecap.BoisOeuvre.Quantite += chantier.VolumeRealise
			myrecap.BoisOeuvre.Benefice += chantier.VolumeRealise * chantier.PUHT
		case "CH":
			myrecap.Chauffage.Quantite += chantier.VolumeRealise
			myrecap.Chauffage.Benefice += chantier.VolumeRealise * chantier.PUHT
		case "PI":
			// ICI PROBLEME, car piquets en stères ou en nb de piquets => calcul faux
			myrecap.Piquets.Quantite += chantier.VolumeRealise
			myrecap.Piquets.Benefice += chantier.VolumeRealise * chantier.PUHT
		case "PL":
			myrecap.Palette.Quantite += chantier.VolumeRealise
			myrecap.Palette.Benefice += chantier.VolumeRealise * chantier.PUHT
		case "PP":
			myrecap.PateAPapier.Quantite += chantier.VolumeRealise
			myrecap.PateAPapier.Benefice += chantier.VolumeRealise * chantier.PUHT
		}
		ug.Recaps[y] = myrecap
	}
	//
	ug.SortedRecapYears = make([]string, 0, len(ug.Recaps))
	for k, _ := range ug.Recaps {
		ug.SortedRecapYears = append(ug.SortedRecapYears, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ug.SortedRecapYears)))
	//
	return nil
}
