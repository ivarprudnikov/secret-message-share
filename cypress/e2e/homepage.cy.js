it('loads the homepage', () => {
    cy.visit('/')
    cy.contains('create self destructing secret messages').should('be.visible')
})