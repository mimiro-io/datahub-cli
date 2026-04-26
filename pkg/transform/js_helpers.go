// Copyright 2026 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0

package transform

const wrapperJavascriptFunction = `
function transform_entities_ex(entities) {
	try {
		return transform_entities(entities);
	} catch (e) {
		throw(e);
	}
}
`

const helperJavascriptFunctions = `
function SetProperty(entity, prefix, name, value) {
	if (entity === null || entity === undefined) {
		return;
	}
	if (entity.Properties === null || entity.Properties === undefined) {
		return;
	}
	entity["Properties"][prefix+":"+name] = value;
}
function GetProperty(entity, prefix, name, defaultValue) {
	if (entity === null || entity === undefined) {
		return defaultValue;
	}
	if (entity.Properties === null || entity.Properties === undefined) {
		return defaultValue;
	}
	var value = entity["Properties"][prefix+":"+name]
	if (value === undefined || value === null) {
		return defaultValue;
	}
	return value;
}
function GetReference(entity, prefix, name, defaultValue) {
	if (entity === null || entity === undefined) {
		return defaultValue;
	}
	if (entity.References === null || entity.References === undefined) {
		return defaultValue;
	}
	var value = entity["References"][prefix+":"+name]
	if (value === undefined || value === null) {
		return defaultValue;
	}
	return value;
}
function AddReference(entity, prefix, name, value) {
	if (entity === null || entity === undefined) {
		return;
	}
	if (entity.References === null || entity.References === undefined) {
		return;
	}
	entity["References"][prefix+":"+name] = value;
}
function GetId(entity) {
	if (entity === null || entity === undefined) {
		return;
	}
	return entity["ID"];
}
function SetId(entity, id) {
	if (entity === null || entity === undefined) {
		return;
	}
	entity.ID = id
}
function SetDeleted(entity, deleted) {
	if (entity === null || entity === undefined) {
		return;
	}
	entity.IsDeleted = deleted
}
function GetDeleted(entity) {
	if (entity === null || entity === undefined) {
		return;
	}
	return entity.IsDeleted;
}
function PrefixField(prefix, field) {
	return prefix + ":" + field;
}
function RenameProperty(entity, originalPrefix, originalName, newPrefix, newName) {
	if (entity === null || entity === undefined) {
		return;
	}
	var value = GetProperty(entity, originalPrefix, originalName);
	SetProperty(entity, newPrefix, newName, value);
	RemoveProperty(entity, originalPrefix, originalName);
}
function RemoveProperty(entity, prefix, name){
	if (entity === null || entity === undefined) {
		return;
	}
	delete entity["Properties"][prefix+":"+name];
}
function NewEntityFrom(entity, addType, copyProps, copyRefs){
	if (entity === null || entity === undefined) {
		return NewEntity();
	}
	let newEntity = NewEntity();
	SetId(newEntity, GetId(entity));
	SetDeleted(newEntity, GetDeleted(entity));
	if (addType){
		let rdf = GetNamespacePrefix("http://www.w3.org/1999/02/22-rdf-syntax-ns#");
		let type = GetReference(entity, rdf, "type");
		if (type != null){
			AddReference(newEntity, rdf, "type", type)
		}
	}
	if (copyProps) {
		for (const [key, value] of Object.entries(entity["Properties"])) {
			newEntity["Properties"][key] = value;
		}
	}
	if (copyRefs) {
		for (const [key, value] of Object.entries(entity["References"])) {
			newEntity["References"][key] = value;
		}
	}
	return newEntity;
}
`
