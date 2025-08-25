document.addEventListener('DOMContentLoaded', () => {
    const orderInput = document.getElementById('orderInput');
    const searchBtn = document.getElementById('searchBtn');
    const resultDiv = document.getElementById('result');

    const getOrder = async () => {
        const orderUID = orderInput.value.trim();
        resultDiv.innerHTML = ''; // Limpiar resultados anteriores

        if (!orderUID) {
            resultDiv.innerHTML = '<div class="error">Please enter an Order UID</div>';
            return;
        }

        try {
            const response = await fetch(`/order/${orderUID}`);
            const data = await response.json();

            if (!response.ok) {
                throw new Error(data.error || 'Order not found');
            }

            // Mostrar JSON formateado
            const pre = document.createElement('pre');
            pre.textContent = JSON.stringify(data, null, 2);
            resultDiv.appendChild(pre);

        } catch (error) {
            resultDiv.innerHTML = `<div class="error">${error.message}</div>`;
        }
    };

    searchBtn.addEventListener('click', getOrder);
    orderInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') getOrder();
    });
});