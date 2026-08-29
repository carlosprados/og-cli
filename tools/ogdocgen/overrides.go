package main

// Corrections to what the documentation says.
//
// The documentation is authoritative about what EXISTS — it is the only source
// for 400-odd signatures — but not always about the details. Every entry here
// was found by type-checking real production artifacts from the sensehat
// organization against generated declarations and getting an error on code that
// works. Each carries its evidence, because an unexplained override is
// indistinguishable from a mistake.
//
// When the documentation is fixed, the corresponding entry can go. The handoff
// in docs/opengate-documentation-handoff.md lists them for that purpose.

// The documentation turned out to need fewer corrections than expected. It
// already writes rest parameters as `logger.trace(...msg)`, which the parser
// now honours — an earlier version dropped the dots and made
// logger.debug('a: ', b) an arity error on working code. That was a parser bug,
// not a documentation one.

// `entity` is an ambient global, not a parameter — which is why nothing here
// widens the is*Action signatures any more.
//
// An earlier version of this file gave isInsertAction, isUpdateAction and
// isPatchAction an optional `entity` parameter, because the live PROVISION_RULE
// in sensehat calls isInsertAction(entity) while the documentation shows
// isInsertAction(). That override treated the documentation as wrong. It was
// not: the platform team confirmed that `entity` is a global of the rule
// function, so these helpers already see the current entity and take no
// argument. `gateway` has the same shape.
//
// typegen already models both as ambient globals — writeEntity emits
// `declare const entity` and `declare const gateway` into every context — so
// the argument in the live rule is vestigial, and the diagnostic on it is
// correct rather than a false positive.

// paramTypes refine a documented parameter to a named TypeScript type.
//
// The documentation states a parameter's type as "string" or "*", which is true
// but loses what makes these declarations useful: a datastream identifier is
// not any string, it is one of the organization's, and an alarm severity is one
// of three words. The named types are declared in the hand-written base
// templates, which cover what the documentation does not describe.
//
// Keyed by "object.method.param", or ".function.param" for a plain function.
var paramTypes = map[string]string{
	// Datastream identifiers: a typo here is the single most common mistake,
	// and the whole reason the datamodel is read at pull time.
	"collection.addDatapoint.datastreamId":  "OGDatastreamID",
	"collection.setFeed.datastreamId":       "OGDatastreamID",
	"collection.getDatastream.datastream":   "OGDatastreamID",
	"collection.getValue.datastream":        "OGDatastreamID",
	".entityValue.datastream":               "OGDatastreamID",
	".entityAt.datastream":                  "OGDatastreamID",
	".entityDate.datastream":                "OGDatastreamID",
	".entitiesValue.datastream":             "OGDatastreamID",
	".getDatastreamFromEntity.datastreamId": "OGDatastreamID",
	".getDatastreamByIdFromDB.datastreamId": "OGDatastreamID",

	// The parameter table says String; the page's own example passes 2 and its
	// prose says "This function returns the same value". A live rule reads a
	// numeric rule parameter through it. The table is wrong, so this widens back
	// to `any` rather than reject working code.
	".getVariableValue.variable": "any",

	// Alarm severity and priority are three-value enumerations; as plain
	// strings, severity: 'HIGH' — a real mistake — would pass.
	"alarm.open.alarmConfig":           "OGAlarmConfig",
	".openAlarm.severity":              "OGSeverity",
	".openAlarm.priority":              "OGPriority",
	".openAlarmWithExtraInfo.severity": "OGSeverity",
	".openAlarmWithExtraInfo.priority": "OGPriority",
}

// skipDecls are declarations the generator must not emit because a
// hand-written base template declares them better, or because they are
// structural rather than API.
var skipDecls = map[string]bool{
	// The base templates declare these with the datastream and sample types the
	// documentation does not describe.
	".entity":  true,
	".gateway": true,
}

// applyOverrides adjusts a bundle before it is emitted.
func applyOverrides(b *Bundle) {
	// Refine parameter types, and drop what the base templates own.
	refine := func(list []Decl) []Decl {
		out := list[:0]
		for _, d := range list {
			if skipDecls[d.Object+"."+d.Name] {
				continue
			}
			for i := range d.Params {
				key := d.Object + "." + d.Name + "." + d.Params[i].Name
				if named, ok := paramTypes[key]; ok {
					d.Params[i].NamedType = named
				}
			}
			out = append(out, d)
		}
		return out
	}
	b.Plain = refine(b.Plain)
	for object, list := range b.Objects {
		b.Objects[object] = refine(list)
	}

}
