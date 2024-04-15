describe('template spec', () => {
  it('creates account', () => {
    cy.visit('/accounts/new')
    cy.contains('Craete your account').should('be.visible')

    const random = Math.random().toString().substr(2, 9)

    cy.get('#username').type('joe-'+random)
    cy.get('#password').type('pass')
    cy.get('#password2').type('pass')
    cy.get('.btn-primary').click()

    cy.contains('Account created').should('be.visible')
  })
})