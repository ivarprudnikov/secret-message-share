Cypress.Commands.add('loginJoe', () => {
    cy.visit('/accounts/login')
    cy.get('#username').type('joe')
    cy.get('#password').type('joe')
    cy.get('.btn-primary').click()
    cy.contains('footer', 'User: joe').should('be.visible')
})
Cypress.Commands.add('loginAlice', () => {
    cy.visit('/accounts/login')
    cy.get('#username').type('alice')
    cy.get('#password').type('alice')
    cy.get('.btn-primary').click()
    cy.contains('footer', 'User: alice').should('be.visible')
})
Cypress.Commands.add('logout', () => {
    cy.get('.logout-link').click()
    cy.contains('footer', 'User:').should('not.exist')
    cy.clearCookies()
})
Cypress.Commands.add('enterPin', (requestPath, pinval) => {
    cy.visit(requestPath)
    // somehow csrf cookie value is not correctly set by cypress after "visit"
    cy.reload() 
    cy.contains('h1', 'Secret message').should('be.visible')
    cy.get('input#pin').should('be.visible').type(pinval)
    cy.get('.btn-primary').click()
})