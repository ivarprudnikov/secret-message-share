describe('stats spec', () => {
  it('anonymous cannot access stats', () => {
    cy.visit('/stats', {
      failOnStatusCode: false,
    })
    cy.url().should('contain', '/accounts/login?failedPath=/stats')
  })
  it('joe cannot access stats', () => {
    cy.loginJoe()
    cy.visit('/stats', {
      failOnStatusCode: false,
    })

    cy.contains('h1', '403: Forbidden').should('be.visible')
  })
  it('admin can access stats', () => {
    cy.loginAdmin()
    cy.visit('/stats', {
      failOnStatusCode: false,
    })

    cy.contains('h1', 'Stats').should('be.visible')

    cy.contains('li', 'Registered accounts').should('be.visible')
    cy.contains('li', 'Active messages').should('be.visible')

  })
})