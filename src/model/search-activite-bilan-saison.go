/*
Calcul d'activités regroupées par saison

@copyright  BDL, Bois du Larzac.
@licence    GPL, conformémént au fichier LICENCE situé à la racine du projet.
@history    2023-04-25 18:44:57+02:00, Thierry Graff : Creation
*/
package model

import (
	"bdl.local/bdl/generic/tiglib"
	"bdl.local/bdl/generic/wilk/werr"
	"github.com/jmoiron/sqlx"
	"time"
)

/* Liste d'activités ayant lieu dans une période donnée */
type ActiviteParSaison struct {
	Datedeb   time.Time
	Datefin   time.Time
	Activites []*Activite
}

type BilanActivitesParSaison struct {
	Datedeb                        time.Time
	Datefin                        time.Time
	TotalActivitesParValo          []*TotalActivitesParValo
}

type TotalActivitesParValo struct {
	TypeValo string
	Volume   float64
	Unite    string
	PrixHT   map[int]float64 // key = id proprio
}

func ComputeBilansActivitesParSaison(db *sqlx.DB, debutSaison string, activites []*Activite) (result []*BilanActivitesParSaison, err error) {
    //
	activitesParSaison, err := ComputeActivitesParSaison(db, debutSaison, activites)
	if err != nil {
		return result, werr.Wrapf(err, "Erreur appel ComputeActivitesParSaison()")
	}
	//
	result = []*BilanActivitesParSaison{}
	for _, activiteParSaison := range activitesParSaison {
		if len(activiteParSaison.Activites) == 0 {
			continue // exclut les saisons sans activités des bilans
		}
		currentRes := BilanActivitesParSaison{
			Datedeb:               activiteParSaison.Datedeb,
			Datefin:               activiteParSaison.Datefin,
			TotalActivitesParValo: []*TotalActivitesParValo{},
		}
		// compute map intermédiaire
		mapValos := map[string]TotalActivitesParValo{}
		for _, activite := range activiteParSaison.Activites {
			valo := activite.TypeValo
			if _, ok := mapValos[valo]; !ok {
				mapValos[valo] = TotalActivitesParValo{TypeValo: valo}
			}
			entry := mapValos[valo]
			entry.Volume += activite.Volume
			entry.Unite = activite.Unite /////////////////// ici faire conversion d'unité pour certaines valos ? ///////////////////
			entry.PrixHT = make(map[int]float64)
			//
			// on répartit systématiquement le prix par proprio
			// (pourrait être évité quand on veut un bilan pour tous les proprios - tant pis)
			err = activite.ComputeSurfaceParProprio(db)
            if err != nil {
                return result, werr.Wrapf(err, "Erreur appel activite.ComputeSurfaceParProprio()")
            }
            for idProprio, surface := range(activite.SurfaceParProprio){
                // ici, prix par proprio proportionnel à la surface
                entry.PrixHT[idProprio] = activite.PrixHT * surface / activite.SurfaceTotale
            }
            //
			mapValos[valo] = entry
		}
		// utilise map intermédiaire pour remplir currentRes
		for valo, total := range mapValos {
			newRes := TotalActivitesParValo{
				TypeValo: valo,
				Volume:   total.Volume,
				Unite:    total.Unite,
				PrixHT:   total.PrixHT,
			}
			currentRes.TotalActivitesParValo = append(currentRes.TotalActivitesParValo, &newRes)
		}
		result = append(result, &currentRes)
	}
	return result, nil
}

func ComputeActivitesParSaison(db *sqlx.DB, debutSaison string, activites []*Activite) (result []*ActiviteParSaison, err error) {
	limites, _, err := ComputeLimitesSaisons(db, debutSaison)
	tiglib.ArrayReverse(limites)
	if err != nil {
		return result, werr.Wrapf(err, "Erreur appel ComputeLimitesSaisons()")
	}
	result = []*ActiviteParSaison{}
	for _, limite := range limites {
		newRes := ActiviteParSaison{Datedeb: limite[0], Datefin: limite[1], Activites: []*Activite{}}
		result = append(result, &newRes)
	}
	for _, activite := range activites {
		for i, _ := range result {
			if (activite.DateActivite.After(result[i].Datedeb) || activite.DateActivite.Equal(result[i].Datedeb)) && activite.DateActivite.Before(result[i].Datefin) {
				result[i].Activites = append(result[i].Activites, activite)
				break
			}
		}
	}
	return result, nil
}
