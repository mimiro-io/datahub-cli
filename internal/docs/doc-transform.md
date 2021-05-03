# Mimiro Datahub CLI - transform

Manage Transformations from cli such as import, export, test and so on. See available Commands.
```
%s
```

## Import code

You use the import command to import your JavaScript code into a Job on the Datahub.

```
mim transform import <job-id> --file=input.json
```

If you omit the job-id, the import command will just base64 encode your function and print that to the console.
When you run the import command, it will also print the transpiled code onto the console, so you can inspect it.

It is important to note that the JavaScript version on the Datahub is basically ES5.1, you should me aware of the 
limitations that this imposes.

**Important** 

If you import your function through the mim import function, you must export it. If you import it manually by 
inserting base64 encoded code directly into the job, you *must* exclude the export.

Example of valid JavaScript:

```javascript
export function transform_entities(entities) {     
    // perform changes of creations  
    return entities;
}

function prefix_field(prefix, field) {
    return prefix + ":" + field;
}
```

Your function must also be named "transform_entities", or it will not be found by the engine.

## Helper functions

A set of helper functions have been implemented in the transform engine to help with writing transforms.

These are:

```javascript
/*
 * Adds a property to an entity
 * 
 * entity - the entity to set the property on
 * prefix - the namespace prefix to use
 * name - field name
 * value - the value to set
 */
function SetProperty(entity, prefix, name, value) {}

/*
 * Returns a given property from an entity
 * 
 * entity - the entity to get the property from
 * prefix - the namespace prefix to use
 * name - field name
 * defaultValue - (optional) if added, this value will be returned if the name field is empty (or entity is empty)
 * 
 * Will return undefined if entity or property is missing
 */
function GetProperty(entity, prefix, name, defaultValue) {}

/*
 * Adds a Reference to an Entity 
 * entity - the entity to add the reference to
 * prefix - the namespace prefix to use
 * name - field name
 * value - the value to set
 */
function AddReference(entity, prefix, name, value) {}

/*
 * Gets the ID of an Entity
 * 
 * entity -  the entity to get the id from
 * 
 * Will return undefined if not set or entity is missing or null
 */
function GetId(entity) {}

/*
 * Sets or changes the ID of an Entity
 * entity - the entity to set the id on
 */
function SetId(entity, id) {}

/*
 * Changes the IsDeleted flag on an Entity
 * 
 * entity - the entity to set the flag on
 * deleted - true or false to set the flag
 */
function SetDeleted(entity, deleted) {}

/*
 * Gets the deleted flag from an Entity
 * 
 * entity - the Entity to get the IsDeleted flag from
 * 
 * If the entity is missing, it will return undefined
 */
function GetDeleted(entity) {}

/*
 * Prefixes a field with a Namespace
 * 
 * prefix - the prefix Namespace
 * field -  the field name to add the prefix to
 * 
 * Returns prefix+":"+field
 */
function PrefixField(prefix, field) {}

/*
 * Renames a field name to the new name, taking prefixes into account
 * 
 * entity - the Entity to change a property on
 * originalPrefix - the original prefix Namespace
 * originalName - the original field name
 * newPrefix - the new prefix Namespace
 * newName - the new field name
 */
function RenameProperty(entity, originalPrefix, originalName, newPrefix, newName) {}

/*
 * Removes a property from the property map
 * 
 * entity - The Entity to remove the property from
 * prefix - the Namespace prefix
 * name - the field name to remove
 */
function RemoveProperty(entity, prefix, name){}

/*
 * Allows to traverse the Graph looking for entities either pointing out from the start or towards the start
 * This is a Go function, so the Go types are added for clarity
 * 
 * startingEntities - a list of entities to start from ie ["ns1:12345"]
 * predicate - the reference to limit the traversal to. Defaults to "*", aka all references.
 * inverse - false to get outgoing references, true to get incoming
 * datasets - a list of datasets to limit the traversal to
 * 
 * This returns a list of list with all the entities that matches the traversal.
 */
function Query(startingEntities /*[]string*/, predicate /*string*/, inverse /*bool*/, datasets /*[]string*/) /* [][]interface{} */ {}

/*
 * Finds a single Entity by its ID
 * This is a Go function, so the Go types are added for clarity
 * 
 * entityId - the id of the Entity to find, namespace must be included
 * datasets - a list of datasets to limit the traversal to, if there are more than 1 Entity with the same ID
 * 
 * Returns a single reference to an Entity, or nil if nothing is found
 */
function FindById(entityId /*string*/, datasets /*[]string*/) /* *server.Entity */ {}

/*
 * Given a Namespace URL, will return the prefix. This is the correct way of looking up namespaces for
 * working with fields and entities, as the Namespace prefixes are different in different envs. When running
 * in the CLI, this will return the correct prefix from the ID it is running in, or if it's new, it will add len +1.
 * This is a Go function, so the Go types are added for clarity
 * 
 * urlExpansion - The expanded URL to get the prefix for
 * 
 * Returns the Namespace prefix
 */
function GetNamespacePrefix(urlExpansion /*string*/) /*string*/ {}

/*
 * This is the cousin of GetNamespacePrefix, and you use this if you are creating a new Dataset within your
 * Transform function. When testing in the CLI, this will just give you a prefix so the function will work,
 * but this is not a stable prefix. In a running DataHub, the prefix and URL will be added as a new Namespace.
 * This is a Go function, so the Go types are added for clarity
 * 
 * urlExpansion - The expanded URL to add
 * 
 * Returns the new Namespace prefix
 */
function AssertNamespacePrefix(urlExpansion /*string*/) /*string*/ {}

/*
 * Logs the content to the DataHub standard logger if running on the Datahub, or to the Console if running
 * in the CLI. If you are running this on a DataHub, on a large dataset, please be careful, as it will drown
 * the logs in log statements.
 * This is a Go function, so the Go types are added for clarity
 * 
 * You can combine this with the ToString() function to get correct logging of Go structs. 
 * 
 * thing - the thing to log
 */
function Log(thing /*interface{}*/) {}

/*
 * Creates a new Entity. If this is returned from the Transform function, this is what's stored in the Dataset.
 * The Entity is empty, so you must set the ID and other properties yourself. You should also create a new Namespace
 * using AssertNamespacePrefix.
 */
function NewEntity() /* *server.Entity*/{}

/*
 * Given an object (obj), will try to pretty format the object.
 * This is implemented in Go in an attempt to show Entities and Go maps into something readable, but there are some 
 * gotcha's here.
 * 
 * For example, in JavaScript there is a difference between undefined (not set) and null (set, but empty), this is not
 * present in Go, so when a nil is encountered, "undefined" is returned. This is chosen because it matches the other
 * functions documented above here, however if you set something to "null" in JavaScript, and call ToString on it, it 
 * will be returned as "undefined". You still has to check for null in your code. For anything returned from one of the 
 * above functions, you have to check for undefined instead.
 * 
 * ToString should mostly be used to pretty print Logs, and you should be careful when converting data types with this
 * function, as there is no guarantee it will be correct.
 */
function ToString(obj /*interface{}*/) /*string*/ {} 

```

