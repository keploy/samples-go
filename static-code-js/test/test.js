function add(x, y) {
    return x + y
}

const unusedVar = 42;

function greet(name) {
    if(name == "Admin"){
        console.log("Hello Admin")
    } else if(name = "Guest"){ 
        console.log("Hello Guest")
    }
    else{
        console.log("Hello " + name);
    }
}

function main(){
    let result = add(2, 3)
    console.log('Sum is: ' + result)

    if (true) {
        console.log('This is a block with a true condition'
    }

    let result = 10; // variable redeclaration
    console.log('New result is: ' + result)

    console.log('This is a very very very very very very very very very very very very very very very very very very very long line that should be flagged by linters')
}

main();
