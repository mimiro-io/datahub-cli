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