describe('messages spec', () => {
  it('joe can see his message in the list', () => {
    cy.visit('/accounts/login')
    cy.get('#username').type('joe')
    cy.get('#password').type('joe')
    cy.get('.btn-primary').click()
    cy.contains('footer', 'User: joe').should('be.visible')
    
    cy.visit('/messages')
    cy.contains('h1', 'Messages').should('be.visible')
    cy.get('.message-row').should('have.length', 1)
  })
  it('alice does not have messages', () => {
    cy.visit('/accounts/login')
    cy.get('#username').type('alice')
    cy.get('#password').type('alice')
    cy.get('.btn-primary').click()
    cy.contains('footer', 'User: alice').should('be.visible')
    
    cy.visit('/messages')
    cy.contains('h1', 'Messages').should('be.visible')
    cy.get('.message-row').should('have.length', 0)
  })
})